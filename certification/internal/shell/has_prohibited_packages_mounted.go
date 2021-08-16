package shell

import (
	"path/filepath"
	"strings"

	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

// HasProhibitedPackages evaluates that the image does not contain prohibited packages,
// which refers to packages that are not redistributable without an appropriate license.
type HasNoProhibitedPackagesMountedCheck struct{}

func (p *HasNoProhibitedPackagesMountedCheck) Validate(image string) (bool, error) {
	pkgList, err := p.getDataToValidate(image)
	if err != nil {
		log.Error("unable to get a list of all packages in the image")
		return false, err
	}

	return p.validate(pkgList)
}

func (p *HasNoProhibitedPackagesMountedCheck) getDataToValidate(dir string) ([]string, error) {
	log.Debugf("Mounted directory: %s", dir)

	db, err := rpmdb.Open(filepath.Join(dir, "var", "lib", "rpm", "Packages"))
	if err != nil {
		return nil, err
	}
	pkgList, err := db.ListPackages()
	if err != nil {
		return nil, err
	}
	pkgs := make([]string, len(pkgList))
	for i, pkg := range pkgList {
		pkgs[i] = pkg.Name
	}
	return pkgs, nil
}

func (p *HasNoProhibitedPackagesMountedCheck) validate(pkgList []string) (bool, error) {
	var prohibitedPackages []string
	for _, pkg := range pkgList {
		_, ok := prohibitedPackageList[pkg]
		if ok {
			prohibitedPackages = append(prohibitedPackages, pkg)
			continue
		}
		for _, name := range prohibitedPackageGlobList {
			if strings.HasPrefix(pkg, name) {
				prohibitedPackages = append(prohibitedPackages, pkg)
				continue
			}
		}
	}

	log.Warn("The number of prohibited package found in the container image: ", len(prohibitedPackages))
	if len(prohibitedPackages) == 0 {
		return true, nil
	}

	log.Warn("found the following prohibited packages: ", prohibitedPackages)
	return false, nil
}

func (p *HasNoProhibitedPackagesMountedCheck) Name() string {
	return "HasNoProhibitedPackagesMounted"
}

func (p *HasNoProhibitedPackagesMountedCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checks to ensure that the image in use does not include prohibited packages, such as Red Hat Enterprise Linux (RHEL) kernel packages.",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *HasNoProhibitedPackagesMountedCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check HasNoProhibitedPackages encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Remove any RHEL packages that are not distributable outside of UBI",
	}
}

// prohibitedPackageList is a list of packages commonly present in the RHEL container images that are not redistributable
// without proper licensing (i.e. packages that are not under the same availability as those found in UBI).
// Implementation detail: Use a map[string]struct{} so that lookups can be done, and determine their existence
// in the map without having to do nested iteration.
// TODO: Confirm these packages are the only packages in immediate scope.
var prohibitedPackageList = map[string]struct{}{
	"grub":                       {},
	"grub2":                      {},
	"kernel":                     {},
	"kernel-core":                {},
	"kernel-debug":               {},
	"kernel-debug-core":          {},
	"kernel-debug-modules":       {},
	"kernel-debug-modules-extra": {},
	"kernel-debug-devel":         {},
	"kernel-devel":               {},
	"kernel-doc":                 {},
	"kernel-modules":             {},
	"kernel-modules-extra":       {},
	"kernel-tools":               {},
	"kernel-tools-libs":          {},
	"kmod-kvdo":                  {},
	"linux-firmware":             {},
}

var prohibitedPackageGlobList = []string{
	"kpatch",
}
