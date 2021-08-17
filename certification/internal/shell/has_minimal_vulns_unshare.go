package shell

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	log "github.com/sirupsen/logrus"
)

// HasMinimalVulnerabilitiesUnshareCheck evaluates the image to confirm that only vulnerabilities
// below the Critical or Important severities, using oscap-podman and a corresponding
// OVAL definition.
type HasMinimalVulnerabilitiesUnshareCheck struct{}

func (p *HasMinimalVulnerabilitiesUnshareCheck) Validate(image string) (bool, error) {
	lines, err := p.scanImage(image)
	if err != nil {
		log.Debugf("Stdout: %s", lines)
		return false, fmt.Errorf("%w: %s", errors.ErrImageScanFailed, err)
	}

	return p.validate(lines)
}

func (p *HasMinimalVulnerabilitiesUnshareCheck) scanImage(image string) ([]string, error) {
	imageScanReport, err := podmanEngine.ScanImage(image)
	if err != nil {
		log.Error("unable to scan the image: ", err)
		return nil, err
	}

	return strings.Split(imageScanReport.Stdout, "\n"), nil
}

func (p *HasMinimalVulnerabilitiesUnshareCheck) validate(lines []string) (bool, error) {

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

func (p *HasMinimalVulnerabilitiesUnshareCheck) Name() string {
	return "HasMinimalVulnerabilitiesUnshare"
}

func (p *HasMinimalVulnerabilitiesUnshareCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking container image does not contain any critical or important security vulnerabilities, as defined at https://access.redhat.com/security/updates/classification.",
		Level:            "good",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *HasMinimalVulnerabilitiesUnshareCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check HasMinimalVulnerabilities encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Update your UBI image to the latest version or update the packages in your image to the latest versions distributed by Red Hat.",
	}
}
