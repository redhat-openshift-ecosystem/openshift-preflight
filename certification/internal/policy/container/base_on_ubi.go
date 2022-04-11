package container

import (
	"context"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	pyxis "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/pyxis"
	log "github.com/sirupsen/logrus"
)

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

func (p *BasedOnUBICheck) Validate(ctx context.Context, imgRef certification.ImageReference) (bool, error) {
	layerHashes, err := p.getImageLayers(ctx, imgRef.ImageInfo)
	if err != nil {
		return false, err
	}

	return p.validate(ctx, layerHashes)
}

// getImageLayers returns the root filesystem DiffIDs of the image.
func (p *BasedOnUBICheck) getImageLayers(ctx context.Context, image cranev1.Image) ([]cranev1.Hash, error) {
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
		log.Error("Error when querying pyxis for uncompressed top layer ids", err)
		return false, err
	}
	if len(certImages) >= 1 {
		return true, nil
	}
	log.Error("No matching layer ids found in pyxis db. Please verify if the image is based on a recent UBI image")
	return false, nil
}

func (p *BasedOnUBICheck) validate(ctx context.Context, layerHashes []cranev1.Hash) (bool, error) {
	hasUBIHash, err := p.certifiedImagesFound(ctx, layerHashes)
	if err != nil {
		log.Error("Unable to verify layer hashes", err)
		return false, err
	}
	if hasUBIHash {
		return true, nil
	}
	return false, nil
}

func (p *BasedOnUBICheck) Name() string {
	return "BasedOnUbi"
}

func (p *BasedOnUBICheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if the container's base image is based upon the Red Hat Universal Base Image (UBI)",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide", // Placeholder
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *BasedOnUBICheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check BasedOnUbi encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Change the FROM directive in your Dockerfile or Containerfile to FROM registry.access.redhat.com/ubi8/ubi",
	}
}
