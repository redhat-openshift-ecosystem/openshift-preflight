// Package replay implements the "replay-submit" workflow: taking the
// artifacts.tar produced by `preflight check container --offline` in a
// disconnected environment, revalidating the BasedOnUbi check (which could
// not reach Pyxis while offline) from a connected host, and submitting the
// finalized results to Red Hat.
package replay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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

// Options configures a replay-submit execution.
type Options struct {
	// ArtifactsTarPath is the path to the artifacts.tar produced by
	// `preflight check container --offline`.
	ArtifactsTarPath string
	// ExtractDir is where the tarball contents are extracted to. If empty, a
	// temporary directory is created and used.
	ExtractDir string

	// Pyxis / submission configuration - mirrors the flags already exposed
	// by `preflight check container`.
	PyxisHost                string
	PyxisAPIToken            string
	CertificationComponentID string
	DockerConfig             string

	// Submit controls whether the finalized results are actually submitted
	// to Red Hat once BasedOnUbi has been revalidated. When false, this is a
	// dry-run: results.json is updated on disk but nothing is sent.
	Submit bool

	// PyxisHTTPClient is the HTTP client used for the BasedOnUbi revalidation
	// query against Pyxis. Defaults to a real *http.Client with a 60s
	// timeout when nil. Exposed primarily so tests can substitute a fake
	// transport instead of hitting the network.
	PyxisHTTPClient pyxis.HTTPClient
}

// Summary describes what happened during a replay-submit execution, for
// reporting back to the user.
type Summary struct {
	ExtractDir      string
	BasedOnUbiPased bool
	Submitted       bool
}

// Run executes the replay-submit workflow:
//  1. Extract artifacts.tar into a working directory.
//  2. Read cert-image.json to recover the image's uncompressed layer DiffIDs.
//  3. Query Pyxis for a certified image (on registry.access.redhat.com) that
//     contains one of those layers - the same check BasedOnUbi performs
//     online.
//  4. Update results.json: remove the old (errored) BasedOnUbi entry and
//     insert a Passed/Failed entry based on step 3, then recompute the
//     top-level Passed field.
//  5. If Submit is set, submit the finalized artifacts to Red Hat using the
//     same submission path `check container --submit` uses.
func Run(ctx context.Context, opts Options) (*Summary, error) {
	logger := logr.FromContextOrDiscard(ctx)

	extractDir := opts.ExtractDir
	if extractDir == "" {
		var err error
		extractDir, err = os.MkdirTemp("", "preflight-replay-submit-")
		if err != nil {
			return nil, fmt.Errorf("could not create temporary extraction directory: %w", err)
		}
	}

	logger.Info("extracting artifacts tarball", "tar", opts.ArtifactsTarPath, "destination", extractDir)
	if err := ExtractArtifactsTar(opts.ArtifactsTarPath, extractDir); err != nil {
		return nil, fmt.Errorf("could not extract %s: %w", opts.ArtifactsTarPath, err)
	}

	certImage, err := readCertImage(extractDir)
	if err != nil {
		return nil, err
	}

	diffIDs, err := diffIDsFromCertImage(certImage)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	logger.Info("revalidated BasedOnUbi", "passed", basedOnUbiPassed)

	results, err := ReadResults(extractDir)
	if err != nil {
		return nil, err
	}

	ApplyBasedOnUbiResult(results, basedOnUbiPassed)

	if err := WriteResults(extractDir, results); err != nil {
		return nil, err
	}
	logger.Info("results.json updated with revalidated BasedOnUbi outcome", "overallPassed", results.Passed)

	summary := &Summary{
		ExtractDir:      extractDir,
		BasedOnUbiPased: basedOnUbiPassed,
	}

	if !opts.Submit {
		logger.Info("submit was not requested; results.json was updated but not sent to Red Hat", "extractDir", extractDir)
		return summary, nil
	}

	if err := submit(ctx, extractDir, opts); err != nil {
		return summary, err
	}
	summary.Submitted = true

	return summary, nil
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

// diffIDsFromCertImage recovers the uncompressed layer DiffIDs of the tested
// image from its previously written cert-image.json, matching what
// internal/policy/container.BasedOnUBICheck.getImageLayers extracts from a
// live image reference.
func diffIDsFromCertImage(certImage *pyxis.CertImage) ([]cranev1.Hash, error) {
	if certImage.ParsedData == nil {
		return nil, fmt.Errorf("%s did not contain parsed_data.uncompressed_layer_sizes; was it written by an older preflight version?", check.DefaultCertImageFilename)
	}

	layers := certImage.ParsedData.UncompressedLayerSizes
	hashes := make([]cranev1.Hash, 0, len(layers))
	for _, layer := range layers {
		hash, err := cranev1.NewHash(layer.LayerID)
		if err != nil {
			return nil, fmt.Errorf("invalid layer id %q in %s: %w", layer.LayerID, check.DefaultCertImageFilename, err)
		}
		hashes = append(hashes, hash)
	}

	if len(hashes) == 0 {
		return nil, fmt.Errorf("no layers found in %s", check.DefaultCertImageFilename)
	}

	return hashes, nil
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

// submit reuses the existing container submission path
// (internal/lib.ContainerCertificationSubmitter) so results are sent to
// Pyxis exactly the same way `check container --submit` would send them,
// but reading from extractDir instead of a live check run's artifacts dir.
func submit(ctx context.Context, extractDir string, opts Options) error {
	artifactWriter, err := artifacts.NewFilesystemWriter(artifacts.WithDirectory(extractDir))
	if err != nil {
		return fmt.Errorf("could not create artifact writer for %s: %w", extractDir, err)
	}
	ctx = artifacts.ContextWithWriter(ctx, artifactWriter)

	pc := lib.NewPyxisClient(ctx, opts.CertificationComponentID, opts.PyxisAPIToken, opts.PyxisHost)
	if pc == nil {
		return fmt.Errorf("pyxis-api-token, certification-component-id, and pyxis-host must all be set to submit results")
	}

	submitter := &lib.ContainerCertificationSubmitter{
		CertificationProjectID: opts.CertificationComponentID,
		Pyxis:                  pc,
		DockerConfig:           opts.DockerConfig,
		PreflightLogFile:       filepath.Join(extractDir, "preflight.log"),
	}

	return submitter.Submit(ctx)
}
