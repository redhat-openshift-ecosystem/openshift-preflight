package container

import (
	"fmt"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

const (
	acceptableLayerMax = 40
)

// UnderLayerMaxCheck ensures that the image has less layers in its assembly than a predefined maximum.
type MaxLayersCheck struct{}

func (p *MaxLayersCheck) Validate(imageRef certification.ImageReference) (bool, error) {
	layers, err := p.getDataToValidate(imageRef.ImageInfo)
	if err != nil {
		return false, err
	}

	return p.validate(layers)
}

func (p *MaxLayersCheck) getDataToValidate(image cranev1.Image) ([]cranev1.Layer, error) {
	return image.Layers()
}

func (p *MaxLayersCheck) validate(layers []cranev1.Layer) (bool, error) {
	log.Debugf("detected %d layers in image", len(layers))
	return len(layers) <= acceptableLayerMax, nil
}

func (p *MaxLayersCheck) Name() string {
	return "LayerCountAcceptable"
}

func (p *MaxLayersCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      fmt.Sprintf("Checking if container has less than %d layers.  Too many layers within the container images can degrade container performance.", acceptableLayerMax),
		Level:            "better",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *MaxLayersCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check LayerCountAcceptable encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Optimize your Dockerfile to consolidate and minimize the number of layers. Each RUN command will produce a new layer. Try combining RUN commands using && where possible.",
	}
}
