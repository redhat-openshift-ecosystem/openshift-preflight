package container

import (
	"context"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	pyxis "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/pyxis"
	log "github.com/sirupsen/logrus"
)

// BasedOnUBICheck evaluates if the provided image is based on the Red Hat Universal Base Image
// by inspecting the contents of the `/etc/os-release` and identifying if the ID is `rhel` and the
// Name value is `Red Hat Enterprise Linux`
type BasedOnUBICheck struct {
	LayerHashCheckEngine layerHashChecker
}

type layerHashChecker interface {
	CheckRedHatLayers(ctx context.Context, layerHashes []cranev1.Hash) ([]pyxis.CertImage, error)
}

func NewBasedOnUbiCheck(layerHashChecker layerHashChecker) *BasedOnUBICheck {
	return &BasedOnUBICheck{LayerHashCheckEngine: layerHashChecker}
}

func (p *BasedOnUBICheck) Validate(imgRef certification.ImageReference) (bool, error) {
	layerHashes, err := p.getImageLayers(imgRef.ImageInfo)
	if err != nil {
		return false, err
	}

	return p.validate(layerHashes)
}

func (p *BasedOnUBICheck) getImageLayers(image cranev1.Image) ([]cranev1.Hash, error) {
	configFile, err := image.ConfigFile()
	if err != nil {
		return nil, err
	}

	return configFile.RootFS.DiffIDs, nil
}

func (p *BasedOnUBICheck) checkRedHatLayers(ctx context.Context, layerHashes []cranev1.Hash) (bool, error) {
	certImages, err := p.LayerHashCheckEngine.CheckRedHatLayers(ctx, layerHashes)
	if err != nil {
		log.Error("Error when querying pyxis for uncompressed top layer ids", err)
	}
	if certImages != nil && len(certImages) >= 1 {
		return true, nil
	}
	return false, nil
}

func (p *BasedOnUBICheck) validate(layerHashes []cranev1.Hash) (bool, error) {
	ctx := context.Background()
	hasUBIHash, err := p.checkRedHatLayers(ctx, layerHashes)
	if err != nil {
		log.Error("Unable to verify layer hashes", err)
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
