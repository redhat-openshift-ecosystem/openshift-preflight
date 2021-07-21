package shell

import (
	"bytes"
	"strings"

	cmdchain "github.com/rainu/go-command-chain"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

type ImageSourceRegistryCheck struct {
}

var approvedRegistries = map[string]string{
	"registry.connect.dev.redhat.com":   "registry.connect.dev.redhat.com",
	"registry.connect.qa.redhat.com":    "registry.connect.qa.redhat.com",
	"registry.connect.stage.redhat.com": "registry.connect.stage.redhat.com",
	"registry.connect.redhat.com":       "registry.connect.redhat.com",
	"registry.redhat.io":                "registry.redhat.io",
	"registry.access.redhat.com":        "registry.access.redhat.com",
}

func (p *ImageSourceRegistryCheck) Validate(bundleImage string) (bool, error) {

	output := &bytes.Buffer{}
	inputContent := strings.NewReader(bundleImage)

	err := cmdchain.Builder().
		WithInput(inputContent).
		Join("cut", "-d", ",", "-f1").
		Join("cut", "-d", "/", "-f1").
		Finalize().WithOutput(output).Run()
	if err != nil {
		log.Error(" Failed to execute cmdchain builder")
		log.Debug(" failed to execute cmdchain builder", err)
		return false, nil
	}

	userRegistry := strings.TrimRight(output.String(), "\n")
	log.Info("Check Image registry for : ", userRegistry)

	if val, ok := approvedRegistries[userRegistry]; ok {
		log.Debugf("Found %s in the list of approved registry", val)
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

func printRegistry(m map[string]string) string {
	output :=""
	for _, value := range m {
		output +=(value+", ")
	}
	return output
}

func (p *ImageSourceRegistryCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message: "ImageSourceRegistry check failed. The image source's registry is not found in the approved registry list.",
		Suggestion: "Approved registries - "+
			printRegistry(approvedRegistries),
	}
}
