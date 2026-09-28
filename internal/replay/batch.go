package replay

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/go-logr/logr"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
)

// DiscoverPlatformTars looks for the per-platform layout produced by
// `preflight check container --offline` against a multi-arch (manifest
// list) image:
//
//	<root>/
//	├── amd64/artifacts.tar
//	├── arm64/artifacts.tar
//	├── ppc64le/artifacts.tar
//	└── s390x/artifacts.tar
//
// It returns a map of platform name (the subdirectory name, e.g. "amd64") to
// the absolute path of that platform's artifacts.tar. An error is returned
// if root cannot be read or if no platform subdirectory contains an
// artifacts.tar.
func DiscoverPlatformTars(root string) (map[string]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", root, err)
	}

	tars := make(map[string]string)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		candidate := filepath.Join(root, entry.Name(), check.DefaultArtifactsTarFileName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			tars[entry.Name()] = candidate
		}
	}

	if len(tars) == 0 {
		return nil, fmt.Errorf("no %s found in any subdirectory of %s; expected a layout like <platform>/%s per architecture",
			check.DefaultArtifactsTarFileName, root, check.DefaultArtifactsTarFileName)
	}

	return tars, nil
}

// BatchSummary aggregates the outcome of running replay-submit against every
// platform discovered under a multi-arch artifacts root.
type BatchSummary struct {
	// Platforms maps platform name -> its Summary, for platforms that
	// completed successfully.
	Platforms map[string]*Summary
	// Errors maps platform name -> the error encountered while processing
	// it. A platform appears in exactly one of Platforms or Errors.
	Errors map[string]error
}

// AllSucceeded reports whether every discovered platform was processed
// without error.
func (b *BatchSummary) AllSucceeded() bool {
	return len(b.Errors) == 0
}

// RunBatch discovers every platform's artifacts.tar under root (see
// DiscoverPlatformTars) and runs the replay-submit workflow against each one
// independently, since each architecture's image has its own layers and
// must be revalidated/submitted on its own - this mirrors how
// `check container` itself processes manifest-list images one platform at a
// time (see platformsToBeProcessed in cmd/preflight/cmd/check_container.go).
//
// Unlike a single Run call, RunBatch does not abort on the first platform
// failure: since platforms are independent, one architecture's Pyxis error
// (or a partner having already fixed and resubmitted one architecture)
// should not block progress on the others. All per-platform outcomes are
// returned in BatchSummary for the caller to report; RunBatch itself only
// returns a non-nil error for setup failures (e.g. root cannot be read).
//
// optsTemplate supplies every Options field except ArtifactsTarPath and
// ExtractDir, which are set per platform. If optsTemplate.ExtractDir is
// set, each platform extracts into a "<ExtractDir>/<platform>" subdirectory
// to avoid collisions; otherwise each gets its own temporary directory.
func RunBatch(ctx context.Context, root string, optsTemplate Options) (*BatchSummary, error) {
	logger := logr.FromContextOrDiscard(ctx)

	platformTars, err := DiscoverPlatformTars(root)
	if err != nil {
		return nil, err
	}

	platforms := make([]string, 0, len(platformTars))
	for platform := range platformTars {
		platforms = append(platforms, platform)
	}
	sort.Strings(platforms) // deterministic order for logs/output

	summary := &BatchSummary{
		Platforms: make(map[string]*Summary, len(platforms)),
		Errors:    make(map[string]error),
	}

	for _, platform := range platforms {
		logger.Info("processing platform", "platform", platform, "tar", platformTars[platform])

		opts := optsTemplate
		opts.ArtifactsTarPath = platformTars[platform]
		if optsTemplate.ExtractDir != "" {
			opts.ExtractDir = filepath.Join(optsTemplate.ExtractDir, platform)
		} else {
			opts.ExtractDir = ""
		}

		platformSummary, err := Run(ctx, opts)
		if err != nil {
			logger.Error(err, "replay-submit failed for platform", "platform", platform)
			summary.Errors[platform] = err
			continue
		}

		summary.Platforms[platform] = platformSummary
	}

	return summary, nil
}
