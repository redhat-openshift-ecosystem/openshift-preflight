package container

import (
	syserrors "errors"
	"os"
	"path/filepath"
	"strings"

	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

// HasProhibitedPackages evaluates that the image does not contain prohibited packages,
// which refers to packages that are not redistributable without an appropriate license.
type HasNoProhibitedPackagesCheck struct{}

func (p *HasNoProhibitedPackagesCheck) Validate(imgRef certification.ImageReference) (bool, error) {
	pkgList, err := p.getDataToValidate(imgRef.ImageFSPath)
	if err != nil {
		log.Error("unable to get a list of all packages in the image")
		return false, err
	}

	return p.validate(pkgList)
}

func (p *HasNoProhibitedPackagesCheck) getDataToValidate(dir string) ([]string, error) {
	// Check for rpmdb.sqlite. If not found, check for Packages
	rpmdirPath := filepath.Join(dir, "var", "lib", "rpm")
	rpmdbPath := filepath.Join(rpmdirPath, "rpmdb.sqlite")

	if _, err := os.Stat(rpmdbPath); syserrors.Is(err, os.ErrNotExist) {
		// rpmdb.sqlite doesn't exist. Fall back to Packages
		rpmdbPath = filepath.Join(rpmdirPath, "Packages")
	}

	db, err := rpmdb.Open(rpmdbPath)
	if err != nil {
		return nil, err
	}
	pkgList, err := db.ListPackages()
	if err != nil {
		return nil, err
	}
	pkgs := make([]string, 0, len(pkgList))
	for _, pkg := range pkgList {
		pkgs = append(pkgs, pkg.Name)
	}
	return pkgs, nil
}

func (p *HasNoProhibitedPackagesCheck) validate(pkgList []string) (bool, error) {
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

	if len(prohibitedPackages) > 0 {
		log.Warn("The number of prohibited package found in the container image: ", len(prohibitedPackages))
		log.Warn("found the following prohibited packages: ", prohibitedPackages)
	}

	return len(prohibitedPackages) == 0, nil
}

func (p *HasNoProhibitedPackagesCheck) Name() string {
	return "HasNoProhibitedPackagesMounted"
}

func (p *HasNoProhibitedPackagesCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checks to ensure that the image in use does not include prohibited packages, such as Red Hat Enterprise Linux (RHEL) kernel packages.",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *HasNoProhibitedPackagesCheck) Help() certification.HelpText {
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
