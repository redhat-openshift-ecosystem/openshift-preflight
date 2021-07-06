package shell

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/utils/file"
	log "github.com/sirupsen/logrus"
)

const (
	ovalFilename = "rhel-8.oval.xml.bz2"
	ovalUrl      = "https://www.redhat.com/security/data/oval/v2/RHEL8/"
	reportFile   = "vuln.html"
)

type HasMinimalVulnerabilitiesCheck struct{}

func (p *HasMinimalVulnerabilitiesCheck) Validate(image string) (bool, error) {

	ovalFileUrl := fmt.Sprintf("%s%s", ovalUrl, ovalFilename)
	dir, err := ioutil.TempDir(".", "oval-")

	if err != nil {
		log.Error("Unable to create temp dir", err)
		return false, err
	}
	log.Debugf("Oval file dir: %s", dir)
	defer os.RemoveAll(dir)

	ovalFilePath := filepath.Join(dir, ovalFilename)
	log.Debugf("Oval file path: %s", ovalFilePath)

	err = file.DownloadFile(ovalFilePath, ovalFileUrl)
	if err != nil {
		log.Error("Unable to download Oval file", err)
		return false, err
	}
	// get the file name
	r := regexp.MustCompile(`(?P<filename>.*).bz2`)
	ovalFilePathDecompressed := filepath.Join(dir, r.FindStringSubmatch(ovalFilename)[1])

	err = file.Unzip(ovalFilePath, ovalFilePathDecompressed)
	if err != nil {
		log.Error("Unable to unzip Oval file: ", err)
		return false, err
	}

	numOfVulns, err := p.numberOfVulnerabilities(image, ovalFilePathDecompressed)
	if err != nil {
		return false, err
	}

	log.Debugf("The number of found vulnerabilities: %d", numOfVulns)

	return numOfVulns == 0, nil
}

// numberOfVulnerabilities takes the `image` and the path to the OVAL file. It runs
// `oscap-podman` against `image`, and parses the output to figure out how many vulnerabilities
//  exist. It also saves results to the vuln.html file in the current directory,
// for the end users' reference.

func (p *HasMinimalVulnerabilitiesCheck) numberOfVulnerabilities(image string, ovalFilePathDecompressed string) (int, error) {

	// run oscap-podman command and save the report to vuln.html
	cmd := exec.Command("oscap-podman", image, "oval", "eval", "--report", reportFile, ovalFilePathDecompressed)
	var out bytes.Buffer
	cmd.Stdout = &out
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		log.Error("unable to execute oscap-podman on the image: ", cmd.Stderr, reportFile)
		return -1, err
	}

	// get the current directory
	path, err := os.Getwd()
	if err != nil {
		log.Error("unable to get the current directory: ", err)
	}

	log.Debugf("The path to vulnerability report: %s/%s ", path, reportFile)

	// use regex to count the number of vulnerabilities

	// oscap-podman writes `Definition oval:com.redhat.<CVE#>: true` to stdout
	// if the vulnerability exist in the container, and `Definition oval:com.redhat.<CVE#>: false`
	// if the vulnerability does not exist
	r := regexp.MustCompile("Definition oval:com.redhat.*: true")
	matches := r.FindAllStringIndex(string(out.String()), -1)

	// count the number of matches
	return len(matches), nil
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
