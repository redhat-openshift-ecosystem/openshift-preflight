package container

import (
	"context"
	"fmt"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/rpm"

	log "github.com/sirupsen/logrus"
)

var _ certification.Check = &HasNoProhibitedPackagesCheck{}

// HasProhibitedPackages evaluates that the image does not contain prohibited packages,
// which refers to packages that are not redistributable without an appropriate license.
type HasNoProhibitedPackagesCheck struct{}

func (p *HasNoProhibitedPackagesCheck) Validate(ctx context.Context, imgRef certification.ImageReference) (bool, error) {
	pkgList, err := p.getDataToValidate(ctx, imgRef.ImageFSPath)
	if err != nil {
		return false, fmt.Errorf("unable to get a list of all packages in the image: %v", err)
	}

	return p.validate(ctx, pkgList)
}

func (p *HasNoProhibitedPackagesCheck) getDataToValidate(ctx context.Context, dir string) ([]string, error) {
	pkgList, err := rpm.GetPackageList(ctx, dir)
	if err != nil {
		return nil, fmt.Errorf("could not get rpm list: %w", err)
	}
	pkgs := make([]string, 0, len(pkgList))
	for _, pkg := range pkgList {
		pkgs = append(pkgs, pkg.Name)
	}
	return pkgs, nil
}

//nolint:unparam // ctx is unused. Keep for future use.
func (p *HasNoProhibitedPackagesCheck) validate(ctx context.Context, pkgList []string) (bool, error) {
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
		log.Debugf("The number of prohibited package found in the container image: %d", len(prohibitedPackages))
		log.Debugf("found the following prohibited packages: %+v", prohibitedPackages)
	}

	return len(prohibitedPackages) == 0, nil
}

func (p *HasNoProhibitedPackagesCheck) Name() string {
	return "HasNoProhibitedPackages"
}

func (p *HasNoProhibitedPackagesCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checks to ensure that the image in use does not include prohibited packages, such as Red Hat Enterprise Linux (RHEL) kernel packages.",
		Level:            "best",
		KnowledgeBaseURL: certDocumentationURL,
		CheckURL:         certDocumentationURL,
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
