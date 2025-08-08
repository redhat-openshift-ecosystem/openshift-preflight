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

// GetPackageList returns the list of packages in the rpm database from
// /var/lib/rpm/rpmdb.sqlite, /var/lib/rpm/Packages or /usr/lib/sysimage/rpm/rpmdb.sqlite.
// If neither exists, this returns an error of type os.ErrNotExists
func GetPackageList(ctx context.Context, basePath string) ([]*rpmdb.PackageInfo, error) {
	rpmdbPaths := []string{
		// Explicitly check /usr/lib/sysimage/rpm. A compatibility symlink from
		// /var/lib/rpm may not necessarily exist.
		filepath.Join(basePath, "usr", "lib", "sysimage", "rpm", "rpmdb.sqlite"),
		filepath.Join(basePath, "var", "lib", "rpm", "rpmdb.sqlite"),
		filepath.Join(basePath, "var", "lib", "rpm", "Packages"),
	}

	var rpmdbPath string
	errs := make([]error, 0, len(rpmdbPaths))
	for _, path := range rpmdbPaths {
		if _, err := os.Stat(path); err != nil {
			errs = append(errs, err)
			continue
		}

		rpmdbPath = path
		break
	}

	if rpmdbPath == "" {
		return nil, fmt.Errorf("could not find rpm db/packages: %v", errors.Join(errs...))
	}

	db, err := rpmdb.Open(rpmdbPath)
	if err != nil {
		return nil, fmt.Errorf("could not open rpm db: %v", err)
	}
	defer db.Close()

	pkgList, err := db.ListPackages()
	if err != nil {
		return nil, fmt.Errorf("could not list packages: %v", err)
	}

	return pkgList, nil
}
