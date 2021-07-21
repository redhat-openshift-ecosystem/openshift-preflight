package shell

import (
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

type ImageSourceRegistryCheck struct {
}

var approvedRegistries = map[string]struct{}{
	"registry.connect.dev.redhat.com":   {},
	"registry.connect.qa.redhat.com":    {},
	"registry.connect.stage.redhat.com": {},
	"registry.connect.redhat.com":       {},
	"registry.redhat.io":                {},
	"registry.access.redhat.com":        {},
}

func (p *ImageSourceRegistryCheck) Validate(bundleImage string) (bool, error) {
	userRegistry := strings.Split(bundleImage, "/")[0]

	log.Info("Check Image registry for : ", userRegistry)

	if _, ok := approvedRegistries[userRegistry]; ok {
		log.Debugf("Found %s in the list of approved registry", userRegistry)
		return true, nil
	}

	log.Info(userRegistry, " not found in approved registry")
	return false, nil
}

func (p *ImageSourceRegistryCheck) Name() string {
	return "OperatorImageSourceRegistryCheck"
}

func (p *ImageSourceRegistryCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Check if image source registry belongs to the approved registry list",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *ImageSourceRegistryCheck) makeRegistryList() string {
	registry := ""
	for key, _ := range approvedRegistries {
		registry += (key + ", ")
	}
	return registry
}

func (p *ImageSourceRegistryCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message: "ImageSourceRegistry check failed. The image source's registry is not found in the approved registry list.",
		Suggestion: "Approved registries - " +
			p.makeRegistryList(),
	}
}
