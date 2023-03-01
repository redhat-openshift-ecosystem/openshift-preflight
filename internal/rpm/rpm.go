package rpm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
	// This pulls in the sqlite dependency
	_ "github.com/glebarez/go-sqlite"
)

// GetPackageList returns the list of packages in the rpm database from either
// /var/lib/rpm/rpmdb.sqlite, or /var/lib/rpm/Packages if the former does not exist.
// If neither exists, this returns an error of type os.ErrNotExists
func GetPackageList(ctx context.Context, basePath string) ([]*rpmdb.PackageInfo, error) {
	rpmdirPath := filepath.Join(basePath, "var", "lib", "rpm")
	rpmdbPath := filepath.Join(rpmdirPath, "rpmdb.sqlite")

	if _, err := os.Stat(rpmdbPath); errors.Is(err, os.ErrNotExist) {
		// rpmdb.sqlite doesn't exist. Fall back to Packages
		rpmdbPath = filepath.Join(rpmdirPath, "Packages")

		// if the fall back path does not exist - this probably isn't a RHEL or UBI based image
		if _, err := os.Stat(rpmdbPath); errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	db, err := rpmdb.Open(rpmdbPath)
	if err != nil {
		return nil, fmt.Errorf("could not open rpm db: %v", err)
	}
	pkgList, err := db.ListPackages()
	if err != nil {
		return nil, fmt.Errorf("could not list packages: %v", err)
	}

	return pkgList, nil
}
