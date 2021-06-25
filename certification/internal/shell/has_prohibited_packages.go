package shell

import (
	"github.com/komish/preflight/certification"
	"github.com/komish/preflight/certification/errors"
	"github.com/sirupsen/logrus"
)

type HasNoProhibitedPackagesPolicy struct{}

func (p *HasNoProhibitedPackagesPolicy) Validate(image string, logger *logrus.Logger) (bool, error) {
	return false, errors.ErrFeatureNotImplemented
}
func (p *HasNoProhibitedPackagesPolicy) Name() string {
	return "HasNoProhibitedPackages"
}
func (p *HasNoProhibitedPackagesPolicy) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checks to ensure that the image in use does not contain prohibited packages.",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		PolicyURL:        "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}

}

func (p *HasNoProhibitedPackagesPolicy) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "The container image should not include Red Hat Enterprise Linux (RHEL) kernel packages.",
		Suggestion: "Remove any RHEL packages that are not distributable outside of UBI",
	}
}

// prohibitedPackageList is a list of packages commonly present in the RHEL contianer images that are not redistributable
// without proper licensing (i.e. packages that are not under the same availability as those found in UBI).
// TODO: Confirm these packages are the only packages in immediate scope.
var prohibitedPackageList []string = []string{
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
