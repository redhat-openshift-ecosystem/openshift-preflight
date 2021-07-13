package shell

import (
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
)

const (
	acceptableLayerMax = 40
)

type UnderLayerMaxCheck struct{}

func (p *UnderLayerMaxCheck) Validate(image string) (bool, error) {
	// TODO: if we're going have the image json on disk already, we should use it here instead of podman inspect-ing.
	layers, err := p.getDataToValidate(image)
	if err != nil {
		return false, err
	}

	return p.validate(layers)
}

func (p *UnderLayerMaxCheck) getDataToValidate(image string) ([]string, error) {
	inspectData, err := podmanEngine.InspectImage(image, cli.ImageInspectOptions{})
	if err != nil {
		return nil, err
	}

	return inspectData.Images[0].RootFS.Layers, nil
}

func (p *UnderLayerMaxCheck) validate(layers []string) (bool, error) {
	log.Debugf("detected %d layers in image", len(layers))
	return len(layers) <= acceptableLayerMax, nil
}

func (p *UnderLayerMaxCheck) Name() string {
	return "LayerCountAcceptable"
}

func (p *UnderLayerMaxCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      fmt.Sprintf("Checking if container has less than %d layers", acceptableLayerMax),
		Level:            "better",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *UnderLayerMaxCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    fmt.Sprintf("Uncompressed container images should have less than %d layers. Too many layers within the container images can degrade container performance.", acceptableLayerMax),
		Suggestion: "Optimize your Dockerfile to consolidate and minimize the number of layers. Each RUN command will produce a new layer. Try combining RUN commands using && where possible.",
	}
}
