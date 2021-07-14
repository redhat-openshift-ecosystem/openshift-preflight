package shell

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	log "github.com/sirupsen/logrus"
)

type HasMinimalVulnerabilitiesCheck struct{}

func (p *HasMinimalVulnerabilitiesCheck) Validate(image string) (bool, error) {
	lines, err := p.getDataToValidate(image)
	if err != nil {
		log.Debugf("Stdout: %s", lines)
		return false, fmt.Errorf("%w: %s", errors.ErrImageScanFailed, err)
	}

	return p.validate(lines)
}

func (p *HasMinimalVulnerabilitiesCheck) getDataToValidate(image string) ([]string, error) {
	imageScanReport, err := podmanEngine.ScanImage(image)
	if err != nil {
		log.Error("unable to scan the image: ", err)
		return nil, err
	}

	return strings.Split(imageScanReport.Stdout, "\n"), nil
}

func (p *HasMinimalVulnerabilitiesCheck) validate(lines []string) (bool, error) {

	// oscap-podman writes `Definition oval:com.redhat.<CVE#>: true` to stdout
	// if the vulnerability exist in the container, and `Definition oval:com.redhat.<CVE#>: false`
	// if the vulnerability does not exist
	r := regexp.MustCompile("Definition oval:com.redhat.*: true")

	numOfVulns := 0
	for _, line := range lines {
		if r.MatchString(line) {
			numOfVulns++
		}
	}
	// count the number of matches
	log.Debugf("The number of found vulnerabilities: %d", numOfVulns)

	return numOfVulns == 0, nil
}

func (p *HasMinimalVulnerabilitiesCheck) Name() string {
	return "HasMinimalVulnerabilities"
}

func (p *HasMinimalVulnerabilitiesCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking for critical or important security vulnerabilites.",
		Level:            "good",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *HasMinimalVulnerabilitiesCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Components in the container image cannot contain any critical or important vulnerabilities, as defined at https://access.redhat.com/security/updates/classification",
		Suggestion: "Update your UBI image to the latest version or update the packages in your image to the latest versions distrubuted by Red Hat.",
	}
}
