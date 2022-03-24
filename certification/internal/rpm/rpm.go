package rpm

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
	preflightErrors "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
)

func GetPackageList(ctx context.Context, basePath string) ([]*rpmdb.PackageInfo, error) {
	rpmdirPath := filepath.Join(basePath, "var", "lib", "rpm")
	rpmdbPath := filepath.Join(rpmdirPath, "rpmdb.sqlite")

	if _, err := os.Stat(rpmdbPath); errors.Is(err, os.ErrNotExist) {
		// rpmdb.sqlite doesn't exist. Fall back to Packages
		rpmdbPath = filepath.Join(rpmdirPath, "Packages")

		// if the fall back path does not exist - this probably isn't a RHEL or UBI based image
		if _, err := os.Stat(rpmdbPath); errors.Is(err, os.ErrNotExist) {
			return nil, preflightErrors.ErrNotSupportedBaseImage
		}
	}

	db, err := rpmdb.Open(rpmdbPath)
	if err != nil {
		return nil, err
	}
	pkgList, err := db.ListPackages()
	if err != nil {
		return nil, err
	}

	return pkgList, nil
}
