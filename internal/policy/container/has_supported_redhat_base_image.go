package container

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/go-logr/logr"
	cranev1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/pyxis"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"
)

var _ check.Check = &HasSupportedRedHatBaseImageCheck{}

const (
	// Pyxis release categories used to determine repository support status.
	releaseCategoryGA         = "Generally Available"
	releaseCategoryDeprecated = "Deprecated"
)

// HasSupportedRedHatBaseImageCheck evaluates whether the Red Hat repository
// containing the image's base layer is still supported and not EOL/Deprecated.
type HasSupportedRedHatBaseImageCheck struct {
	CertifiedRepositoriesFinder certifiedRepositoriesFinder
	now                         func() time.Time
	buildDate                   time.Time
}

type certifiedRepositoriesFinder interface {
	CertifiedRepositoriesForLayers(ctx context.Context, uncompressedLayerHashes []cranev1.Hash) ([]pyxis.CertRepository, error)
}

func NewHasSupportedRedHatBaseImageCheck(certifiedRepositoriesFinder certifiedRepositoriesFinder, now func() time.Time) *HasSupportedRedHatBaseImageCheck {
	if now == nil {
		now = time.Now
	}

	// An empty or unparseable BuildDate (e.g. development builds via
	// plain `go build`) leaves buildDate as the zero value, which
	// disables the clock sanity check.
	buildDate, _ := time.Parse(time.RFC3339, version.BuildDate())

	return &HasSupportedRedHatBaseImageCheck{
		CertifiedRepositoriesFinder: certifiedRepositoriesFinder,
		now:                         now,
		buildDate:                   buildDate,
	}
}

func (p *HasSupportedRedHatBaseImageCheck) Validate(ctx context.Context, imgRef image.ImageReference) (result bool, err error) {
	layerHashes, err := getImageLayers(imgRef.ImageInfo)
	if err != nil {
		return false, fmt.Errorf("could not get image layers: %w", err)
	}

	return p.validate(ctx, layerHashes)
}

func (p *HasSupportedRedHatBaseImageCheck) validate(ctx context.Context, layerHashes []cranev1.Hash) (result bool, err error) {
	logger := logr.FromContextOrDiscard(ctx)

	certRepositories, err := p.CertifiedRepositoriesFinder.CertifiedRepositoriesForLayers(ctx, layerHashes)
	if err != nil {
		return false, fmt.Errorf("pyxis query for certified registries containing uncompressed top layers ids %+q failed: %w", layerHashes, err)
	}

	if len(certRepositories) == 0 {
		// No Red Hat repository matched the image layers.
		return false, nil
	}

	// Keep the current timestamp in UTC to match the build timestamp and
	// keep log/error output consistent.
	now := p.now().UTC()
	logger.Info("evaluating repository end of life dates against current time", "now", now)

	// The system clock reads before the time the preflight binary
	// was built. The EOL dates cannot be trusted; fail the check.
	if !p.buildDate.IsZero() && now.Before(p.buildDate) {
		return false, fmt.Errorf("cannot evaluate repository end of life dates: system clock (%s) reads before binary build time (%s)",
			now.Format(time.RFC3339), p.buildDate.Format(time.RFC3339))
	}

	for _, registry := range certRepositories {
		// A supported repository is not EOL, not Deprecated, and is Generally Available.
		notEOL := registry.EOLDate == nil || now.Before(*registry.EOLDate)

		if notEOL && slices.Contains(registry.ReleaseCategories, releaseCategoryGA) &&
			!slices.Contains(registry.ReleaseCategories, releaseCategoryDeprecated) {
			// One supported repository is sufficient.
			return true, nil
		}
	}

	// All matched repositories are EOL or not Generally Available.
	return false, nil
}

func (p *HasSupportedRedHatBaseImageCheck) Name() string {
	return "HasSupportedRedHatBaseImage"
}

func (p *HasSupportedRedHatBaseImageCheck) RequiredFilePatterns() []string {
	return nil
}

func (p *HasSupportedRedHatBaseImageCheck) Metadata() check.Metadata {
	return check.Metadata{
		Description:      "Checking if the container's Red Hat base image repository is supported",
		Level:            check.LevelWarn,
		KnowledgeBaseURL: certDocumentationURL,
		CheckURL:         certDocumentationURL,
	}
}

func (p *HasSupportedRedHatBaseImageCheck) Help() check.HelpText {
	return check.HelpText{
		Message:    "Check HasSupportedRedHatBaseImage encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Use a supported (non-Deprecated) Red Hat base image from https://catalog.redhat.com/software/containers/",
	}
}
