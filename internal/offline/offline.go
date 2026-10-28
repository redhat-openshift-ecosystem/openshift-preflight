// Package offline implements submit-offline-artifacts: revalidate offline
// checks from a connected host and submit finalized results to Red Hat.
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

// layerHashChecker is satisfied by *pyxis.pyxisClient. Declared so tests can
// substitute a fake without constructing a full Pyxis client.
type layerHashChecker interface {
	CertifiedImagesContainingLayers(ctx context.Context, uncompressedLayerHashes []cranev1.Hash) ([]pyxis.CertImage, error)
}

// projectFetcher is satisfied by *pyxis.pyxisClient for isScratchProject tests.
type projectFetcher interface {
	GetProject(ctx context.Context) (*pyxis.CertProject, error)
}

// deferredRevalidator revalidates one deferred check against Pyxis.
type deferredRevalidator func(ctx context.Context, layers layerHashChecker, diffIDs []cranev1.Hash) (bool, error)

// deferredCheck is a check that needs Pyxis and cannot finish offline.
type deferredCheck struct {
	check      check.Check
	revalidate deferredRevalidator
}

// deferredChecks are revalidated on submit-offline-artifacts, or removed
// entirely when the project has a scratch exception. Add new entries here
// (e.g. HasSupportedRedHatBaseImage / EOL) as they gain offline deferral.
var deferredChecks = []deferredCheck{
	{check: basedOnUbiCheck, revalidate: revalidateBasedOnUbi},
}

// Options configures a submit-offline-artifacts execution.
type Options struct {
	// ArtifactsTarPath is the path to the artifacts.tar produced by
	// `preflight check container --offline`.
	ArtifactsTarPath string

	// LogFile is this run's --logfile/PFLT_LOGFILE (not the extracted one).
	// Defaults to "preflight.log" if empty.
	LogFile string

	// Pyxis / submission configuration - mirrors the flags already exposed
	// by `preflight check container`.
	PyxisHost                string
	PyxisAPIToken            string
	CertificationComponentID string

	// PyxisHTTPClient is used for Pyxis revalidation queries. Defaults to a
	// 60s *http.Client when nil; tests may inject a fake transport.
	PyxisHTTPClient pyxis.HTTPClient

	// PyxisClient is used to submit the finalized results. Defaults to
	// lib.NewPyxisClient(...) when nil; exposed for tests to fake.
	PyxisClient lib.PyxisClient
}

// CheckOutcome records the pass/fail result of a single check that was
// revalidated during submit-offline-artifacts.
type CheckOutcome struct {
	Name   string
	Passed bool
}

// Summary describes what happened during a submit-offline-artifacts execution, for
// reporting back to the user.
type Summary struct {
	ExtractDir string
	Submitted  bool

	// ScratchExempt is true when CertProject.ScratchProject() holds;
	// deferredChecks are then removed instead of revalidated.
	ScratchExempt bool

	// Revalidated is the outcome of each deferred check revalidated this run
	// (empty when ScratchExempt, since those checks were removed).
	Revalidated []CheckOutcome
}

// Run extracts artifacts.tar, revalidates offline-incomplete checks, updates
// results.json, submits, then always removes the temp extraction directory.
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

// run is Run without cleaning up extractDir, so tests can inspect artifacts.
func run(ctx context.Context, opts Options) (extractDir string, summary *Summary, err error) {
	logger := logr.FromContextOrDiscard(ctx)

	// Fresh temp dir (not the tarball's dir) so a failed rerun can't clobber
	// a previously-validated results.json.
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

	// Offline can't see policy exceptions, so deferred checks always err for
	// scratch projects under the default policy; remove or revalidate here.
	scratchExempt, err := isScratchProject(ctx, pyxisClient)
	if err != nil {
		return extractDir, nil, err
	}

	results, err := ReadResults(extractDir)
	if err != nil {
		return extractDir, nil, err
	}

	summary = &Summary{
		ExtractDir:    extractDir,
		ScratchExempt: scratchExempt,
	}

	if scratchExempt {
		for _, dc := range deferredChecks {
			logger.Info("scratch exception; removing deferred check instead of revalidating",
				"check", dc.check.Name())
			RemoveCheckResult(results, dc.check)
		}
	} else {
		for _, dc := range deferredChecks {
			passed, revalidateErr := dc.revalidate(ctx, pyxisClient, diffIDs)
			if revalidateErr != nil {
				return extractDir, nil, revalidateErr
			}
			logger.Info("revalidated deferred check", "check", dc.check.Name(), "passed", passed)
			ApplyCheckResult(results, dc.check, passed)
			summary.Revalidated = append(summary.Revalidated, CheckOutcome{
				Name:   dc.check.Name(),
				Passed: passed,
			})
		}
	}

	if err := WriteResults(extractDir, results); err != nil {
		//coverage:ignore
		return extractDir, nil, err
	}
	logger.Info("results.json updated with deferred check outcomes",
		"overallPassed", results.Passed,
		"scratchExempt", scratchExempt,
	)

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

// diffIDsFromCertImage recovers uncompressed layer DiffIDs from raw_config.
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

// isScratchProject reports whether the project has a scratch exception.
func isScratchProject(ctx context.Context, client projectFetcher) (bool, error) {
	certProject, err := client.GetProject(ctx)
	if err != nil {
		return false, fmt.Errorf("could not retrieve certification project: %w", err)
	}
	if certProject == nil {
		return false, fmt.Errorf("no certification project was returned from pyxis")
	}

	return certProject.ScratchProject(), nil
}

// revalidateBasedOnUbi is the deferredRevalidator for BasedOnUbi.
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
