package shell

import (
	"bufio"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
)

type HasNoProhibitedPackagesCheck struct{}

func (p *HasNoProhibitedPackagesCheck) Validate(image string) (bool, error) {
	line, err := p.getDataToValidate(image)
	if err != nil {
		log.Error("unable to get a list of all packages in the image")
		return false, err
	}

	return p.validate(line)

}

func (p *HasNoProhibitedPackagesCheck) getDataToValidate(image string) (string, error) {
	runOpts := cli.ImageRunOptions{
		EntryPoint:     "rpm",
		EntryPointArgs: []string{"-qa", "--queryformat", "%{NAME}\n"},
		LogLevel:       "debug",
		Image:          image,
	}
	runReport, err := podmanEngine.Run(runOpts)
	if err != nil {
		log.Error("unable to get a list of all packages in the image, error: ", err)
		log.Debugf("Stdout: %s", runReport.Stdout)
		log.Debugf("Stderr: %s", runReport.Stderr)
		return "", err
	}
	return runReport.Stdout, nil
}

func (p *HasNoProhibitedPackagesCheck) validate(line string) (bool, error) {
	scanner := bufio.NewScanner(strings.NewReader(line))
	var prohibitedPackages []string
	for scanner.Scan() {
		for _, pkg := range prohibitedPackageList {
			if pkg == scanner.Text() {
				prohibitedPackages = append(prohibitedPackages, pkg)
			}
		}
	}
	log.Warn("The number of prohibited package found in the container image: ", len(prohibitedPackages))
	if len(prohibitedPackages) > 0 {
		log.Warn("found the following prohibited packages: ", prohibitedPackages)
	}

	return len(prohibitedPackages) == 0, nil
}

func (p *HasNoProhibitedPackagesCheck) Name() string {
	return "HasNoProhibitedPackages"
}
func (p *HasNoProhibitedPackagesCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checks to ensure that the image in use does not contain prohibited packages.",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *HasNoProhibitedPackagesCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "The container image should not include Red Hat Enterprise Linux (RHEL) kernel packages.",
		Suggestion: "Remove any RHEL packages that are not distributable outside of UBI",
	}
}

// prohibitedPackageList is a list of packages commonly present in the RHEL contianer images that are not redistributable
// without proper licensing (i.e. packages that are not under the same availability as those found in UBI).
// TODO: Confirm these packages are the only packages in immediate scope.
var prohibitedPackageList = []string{
	"grub",
	"grub2",
	"kernel",
	"kernel-core",
	"kernel-debug",
	"kernel-debug-core",
	"kernel-debug-modules",
	"kernel-debug-modules-extra",
	"kernel-debug-devel",
	"kernel-devel",
	"kernel-doc",
	"kernel-modules",
	"kernel-modules-extra",
	"kernel-tools",
	"kernel-tools-libs",
	"kmod-kvdo",
	"kpatch*",
	"linux-firmware",
}
