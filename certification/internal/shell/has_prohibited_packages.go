package shell

import (
	"bufio"
	"bytes"
	"os/exec"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/sirupsen/logrus"
)

type HasNoProhibitedPackagesCheck struct{}

func (p *HasNoProhibitedPackagesCheck) Validate(image string, logger *logrus.Logger) (bool, error) {
	stdouterr, err := exec.Command("podman", "run", "-it", "--rm", "--entrypoint", "rpm", image, "-qa", "--queryformat", "%{NAME}\n").CombinedOutput()
	if err != nil {
		logger.Error("unable to get a list of all packages in the image")
		return false, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(stdouterr))
	for scanner.Scan() {
		for _, pkg := range prohibitedPackageList {
			if pkg == scanner.Text() {
				logger.Warn("found a prohibited package in the container image: ", pkg)
				return false, nil
			}
		}
	}

	return true, nil
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
