// Package offline implements the "submit-offline-artifacts" workflow:
// revalidating checks that couldn't complete offline (e.g. BasedOnUbi) from
// a connected host, and submitting the finalized results to Red Hat.
package offline

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-logr/logr"
	cranev1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/lib"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/pyxis"
)

// layerHashChecker is satisfied by *pyxis.pyxisClient. Declared locally so
// this package doesn't need to depend on internal/policy/container.
type layerHashChecker interface {
	CertifiedImagesContainingLayers(ctx context.Context, uncompressedLayerHashes []cranev1.Hash) ([]pyxis.CertImage, error)
}

// Options configures a submit-offline-artifacts execution.
type Options struct {
	// ArtifactsTarPath is the path to the artifacts.tar produced by
	// `preflight check container --offline`.
	ArtifactsTarPath string

	// LogFile is this invocation's own --logfile/PFLT_LOGFILE, submitted
	// alongside the results instead of the preflight.log extracted from
	// ArtifactsTarPath. Defaults to "preflight.log" if empty.
	LogFile string

	// Pyxis / submission configuration - mirrors the flags already exposed
	// by `preflight check container`.
	PyxisHost                string
	PyxisAPIToken            string
	CertificationComponentID string

	// PyxisHTTPClient is used for check revalidation queries against Pyxis
	// (e.g. BasedOnUbi). Defaults to a real *http.Client with a 60s timeout
	// when nil; exposed so tests can substitute a fake transport.
	PyxisHTTPClient pyxis.HTTPClient

	// PyxisClient is used to submit the finalized results. Defaults to
	// lib.NewPyxisClient(...) when nil; exposed for tests to fake.
	PyxisClient lib.PyxisClient
}

// Summary describes what happened during a submit-offline-artifacts execution, for
// reporting back to the user.
type Summary struct {
	ExtractDir       string
	BasedOnUbiPassed bool
	Submitted        bool
}

// Run extracts artifacts.tar, revalidates the checks that couldn't complete
// offline (currently BasedOnUbi) against Pyxis, updates results.json with
// the outcome, and submits the finalized artifacts the same way
// `check container --submit` does. The temporary extraction directory is
// always removed before Run returns, regardless of outcome.
func Run(ctx context.Context, opts Options) (*Summary, error) {
	extractDir, summary, err := run(ctx, opts)
	if extractDir != "" {
		if rmErr := os.RemoveAll(extractDir); rmErr != nil {
			//coverage:ignore
			logr.FromContextOrDiscard(ctx).Error(rmErr, "unable to clean up extraction directory", "extractDir", extractDir)
		}
	}
	return summary, err
}

// run implements Run's logic without cleaning up extractDir, so tests can
// inspect the extracted/updated artifacts after a run completes. Run is a
// thin wrapper around this that always removes extractDir afterward.
func run(ctx context.Context, opts Options) (extractDir string, summary *Summary, err error) {
	logger := logr.FromContextOrDiscard(ctx)

	// Always a fresh temp dir, never the tarball's own directory, so a
	// failed rerun can't clobber a previously-validated results.json.
	extractDir, err = os.MkdirTemp("", "preflight-submit-offline-artifacts-")
	if err != nil {
		//coverage:ignore
		return "", nil, fmt.Errorf("could not create temporary extraction directory: %w", err)
	}

	logger.Info("extracting artifacts tarball", "tar", opts.ArtifactsTarPath, "destination", extractDir)
	if err := extractArtifactsTar(opts.ArtifactsTarPath, extractDir); err != nil {
		return extractDir, nil, fmt.Errorf("could not extract %s: %w", opts.ArtifactsTarPath, err)
	}

	certImage, err := readCertImage(extractDir)
	if err != nil {
		return extractDir, nil, err
	}

	diffIDs, err := diffIDsFromCertImage(certImage)
	if err != nil {
		return extractDir, nil, err
	}
	logger.V(log.DBG).Info("recovered image layers from cert-image.json", "layerCount", len(diffIDs))

	httpClient := opts.PyxisHTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}

	pyxisClient := pyxis.NewPyxisClient(
		opts.PyxisHost,
		opts.PyxisAPIToken,
		opts.CertificationComponentID,
		httpClient,
	)

	basedOnUbiPassed, err := revalidateBasedOnUbi(ctx, pyxisClient, diffIDs)
	if err != nil {
		return extractDir, nil, err
	}
	logger.Info("revalidated BasedOnUbi", "passed", basedOnUbiPassed)

	results, err := ReadResults(extractDir)
	if err != nil {
		return extractDir, nil, err
	}

	ApplyCheckResult(results, basedOnUbiCheck, basedOnUbiPassed)

	if err := WriteResults(extractDir, results); err != nil {
		//coverage:ignore
		return extractDir, nil, err
	}
	logger.Info("results.json updated with revalidated BasedOnUbi outcome", "overallPassed", results.Passed)

	summary = &Summary{
		ExtractDir:       extractDir,
		BasedOnUbiPassed: basedOnUbiPassed,
	}

	if err := submit(ctx, extractDir, opts); err != nil {
		return extractDir, summary, err
	}
	summary.Submitted = true

	return extractDir, summary, nil
}

// readCertImage reads and unmarshals cert-image.json from dir.
func readCertImage(dir string) (*pyxis.CertImage, error) {
	b, err := os.ReadFile(filepath.Join(dir, check.DefaultCertImageFilename))
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", check.DefaultCertImageFilename, err)
	}

	var certImage pyxis.CertImage
	if err := json.Unmarshal(b, &certImage); err != nil {
		return nil, fmt.Errorf("could not unmarshal %s: %w", check.DefaultCertImageFilename, err)
	}

	return &certImage, nil
}

// diffIDsFromCertImage recovers the image's uncompressed layer DiffIDs by
// parsing cert-image.json's raw_config field, the same layer data a live
// image's ConfigFile() would provide online.
func diffIDsFromCertImage(certImage *pyxis.CertImage) ([]cranev1.Hash, error) {
	if certImage.RawConfig == "" {
		return nil, fmt.Errorf("%s did not contain raw_config", check.DefaultCertImageFilename)
	}

	configFile, err := cranev1.ParseConfigFile(strings.NewReader(certImage.RawConfig))
	if err != nil {
		return nil, fmt.Errorf("could not parse raw_config from %s: %w", check.DefaultCertImageFilename, err)
	}

	if len(configFile.RootFS.DiffIDs) == 0 {
		return nil, fmt.Errorf("no layers found in %s's raw_config.rootfs.diff_ids", check.DefaultCertImageFilename)
	}

	return configFile.RootFS.DiffIDs, nil
}

// revalidateBasedOnUbi mirrors internal/policy/container.BasedOnUBICheck.validate:
// an image is considered based on UBI if Pyxis returns at least one certified
// image on registry.access.redhat.com containing one of the given layers.
func revalidateBasedOnUbi(ctx context.Context, checker layerHashChecker, diffIDs []cranev1.Hash) (bool, error) {
	certImages, err := checker.CertifiedImagesContainingLayers(ctx, diffIDs)
	if err != nil {
		return false, fmt.Errorf("pyxis query for uncompressed top layer ids failed: %w", err)
	}

	return len(certImages) >= 1, nil
}

// submit reuses lib.ContainerCertificationSubmitter so results are sent the
// same way `check container --submit` sends them, reading from extractDir.
func submit(ctx context.Context, extractDir string, opts Options) error {
	artifactWriter, err := artifacts.NewFilesystemWriter(artifacts.WithDirectory(extractDir))
	if err != nil {
		//coverage:ignore
		return fmt.Errorf("could not create artifact writer for %s: %w", extractDir, err)
	}
	ctx = artifacts.ContextWithWriter(ctx, artifactWriter)

	pc := opts.PyxisClient
	if pc == nil {
		pc = lib.NewPyxisClient(ctx, opts.CertificationComponentID, opts.PyxisAPIToken, opts.PyxisHost)
	}

	logFile := opts.LogFile
	if logFile == "" {
		logFile = "preflight.log"
	}

	submitter := &lib.ContainerCertificationSubmitter{
		CertificationProjectID: opts.CertificationComponentID,
		Pyxis:                  pc,
		PreflightLogFile:       logFile,
	}

	return submitter.Submit(ctx)
}
