package shell

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
)

type HasRequiredLabelsCheck struct{}

func (p *HasRequiredLabelsCheck) Validate(image string) (bool, error) {
	// TODO: if we're going have the image json on disk already, we should use it here instead of podman inspect-ing.
	labels, err := p.getDataForValidate(image)
	if err != nil {
		return false, err
	}

	return p.validate(labels)
}

func (p *HasRequiredLabelsCheck) getDataForValidate(image string) (map[string]string, error) {
	inspectReport, err := podmanEngine.InspectImage(image, cli.ImageInspectOptions{})
	if err != nil {
		log.Error("unable to execute inspect on the image: ", err)
		return nil, err
	}
	return inspectReport.Images[0].Config.Labels, nil
}

func (p *HasRequiredLabelsCheck) validate(labels map[string]string) (bool, error) {
	requiredLabels := []string{"name", "vendor", "version", "release", "summary", "description"}
	missingLabels := []string{}
	for _, label := range requiredLabels {
		if labels[label] == "" {
			missingLabels = append(missingLabels, label)
		}
	}

	if len(missingLabels) > 0 {
		log.Warn("expected labels are missing:", missingLabels)
		return false, nil
	}

	return true, nil
}

func (p *HasRequiredLabelsCheck) Name() string {
	return "HasRequiredLabel"
}

func (p *HasRequiredLabelsCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if the container's base image is based on UBI",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *HasRequiredLabelsCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "It is recommened that your image be based upon the Red Hat Universal Base Image (UBI)",
		Suggestion: "Change the FROM directive in your Dockerfile or Containerfile to FROM registry.access.redhat.com/ubi8/ubi",
	}
}
