package shell

import (
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
)

type HasLicenseCheck struct{}

func (p *HasLicenseCheck) Validate(image string) (bool, error) {
	line, err := p.getDataToValidate(image)
	if err != nil {
		return false, err
	}
	return p.validate(line)
}

func (p *HasLicenseCheck) getDataToValidate(image string) (string, error) {
	runOpts := cli.ImageRunOptions{
		EntryPoint:     "ls",
		EntryPointArgs: []string{"-A", "/licenses"},
		LogLevel:       "debug",
		Image:          image,
	}
	runReport, err := podmanEngine.Run(runOpts)
	if err != nil {
		ok, _ := p.validate(runReport.Stdout)
		if ok {
			log.Error("some error attempting to identify if /licenses container the license: ", err)
			log.Debugf("Stdout: %s", runReport.Stdout)
			log.Debugf("Stderr: %s", runReport.Stderr)
			return "", err
		}

	}
	return runReport.Stdout, nil
}

func (p *HasLicenseCheck) validate(line string) (bool, error) {

	return !strings.Contains(line, "No such file or directory") && line != "", nil
}

func (p *HasLicenseCheck) Name() string {
	return "HasLicense"
}

func (p *HasLicenseCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if terms and conditions applicable to the software including open source licensing information are present. The license must be at /licenses",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *HasLicenseCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check HasLicense encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Create a directory named /licenses and include all relevant licensing and/or terms and conditions as text file(s) in that directory.",
	}
}
