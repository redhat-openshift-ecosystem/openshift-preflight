package container

import (
	"context"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/pyxis"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
)

var _ check.Check = &BasedOnUBICheck{}

// BasedOnUBICheck evaluates if the provided image is based on the Red Hat Universal Base Image.
type BasedOnUBICheck struct {
	LayerHashCheckEngine layerHashChecker
}

type layerHashChecker interface {
	CertifiedImagesContainingLayers(ctx context.Context, uncompressedLayerHashes []cranev1.Hash) ([]pyxis.CertImage, error)
}

func NewBasedOnUbiCheck(layerHashChecker layerHashChecker) *BasedOnUBICheck {
	return &BasedOnUBICheck{LayerHashCheckEngine: layerHashChecker}
}

func (p *BasedOnUBICheck) Validate(ctx context.Context, imgRef image.ImageReference) (bool, error) {
	layerHashes, err := p.getImageLayers(imgRef.ImageInfo)
	if err != nil {
		return false, fmt.Errorf("could not get image layers: %v", err)
	}

	return p.validate(ctx, layerHashes)
}

// getImageLayers returns the root filesystem DiffIDs of the image.
func (p *BasedOnUBICheck) getImageLayers(image cranev1.Image) ([]cranev1.Hash, error) {
	configFile, err := image.ConfigFile()
	if err != nil {
		return nil, err
	}

	return configFile.RootFS.DiffIDs, nil
}

// certifiedImagesFound checks to make sure images exist in Red Hat Pyxis containing the uncompressed
// top layer IDs of the image under test.
func (p *BasedOnUBICheck) certifiedImagesFound(ctx context.Context, layerHashes []cranev1.Hash) (bool, error) {
	certImages, err := p.LayerHashCheckEngine.CertifiedImagesContainingLayers(ctx, layerHashes)
	if err != nil {
		return false, fmt.Errorf("pyxis query for uncompressed top layers ids %+q failed: %w", layerHashes, err)
	}
	if len(certImages) >= 1 {
		return true, nil
	}
	return false, nil
}

func (p *BasedOnUBICheck) validate(ctx context.Context, layerHashes []cranev1.Hash) (bool, error) {
	hasUBIHash, err := p.certifiedImagesFound(ctx, layerHashes)
	if err != nil {
		return false, fmt.Errorf("unable to verify layer hashes: %v", err)
	}
	if hasUBIHash {
		return true, nil
	}
	return false, nil
}

func (p *BasedOnUBICheck) Name() string {
	return "BasedOnUbi"
}

func (p *BasedOnUBICheck) Metadata() check.Metadata {
	return check.Metadata{
		Description:      "Checking if the container's base image is based upon the Red Hat Universal Base Image (UBI)",
		Level:            "best",
		KnowledgeBaseURL: certDocumentationURL,
		CheckURL:         certDocumentationURL,
	}
}

func (p *BasedOnUBICheck) Help() check.HelpText {
	return check.HelpText{
		Message:    "Check BasedOnUbi encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Change the FROM directive in your Dockerfile or Containerfile, for the latest list of images and details refer to: https://catalog.redhat.com/software/base-images",
	}
}

func (p *BasedOnUBICheck) RequiredFilePatterns() []string {
	return nil
}
