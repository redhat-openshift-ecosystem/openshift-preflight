package container

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-logr/logr"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"
)

const (
	licensePath         = "/licenses"
	minLicenseFileCount = 1
)

var errLicensesNotADir = errors.New("licenses is not a directory")

// canonicalMountRoot returns a path comparable to filepath.EvalSymlinks results
// (e.g. /var/... on macOS resolves to /private/var/...).
func canonicalMountRoot(mountedPath string) string {
	c := filepath.Clean(mountedPath)
	if r, err := filepath.EvalSymlinks(c); err == nil && r != "" {
		return filepath.Clean(r)
	}
	return c
}

// pathWithinMount reports whether resolved is the mount root or a path strictly
// beneath it (prefix boundary safe, e.g. /tmp/img does not contain /tmp/image).
func pathWithinMount(mount, resolved string) bool {
	mount = filepath.Clean(mount)
	resolved = filepath.Clean(resolved)
	if resolved == mount {
		return true
	}
	rel, err := filepath.Rel(mount, resolved)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// resolvedDirEntry wraps a WalkDir DirEntry so Info() returns metadata from
// os.Stat (the followed target) instead of the symlink's DirEntry metadata.
type resolvedDirEntry struct {
	fs.DirEntry
	fi fs.FileInfo
}

func (r resolvedDirEntry) Info() (fs.FileInfo, error) {
	return r.fi, nil
}

var _ check.Check = &HasLicenseCheck{}

// HasLicenseCheck evaluates that the image contains a license definition available at
// /licenses.
type HasLicenseCheck struct{}

func (p *HasLicenseCheck) Validate(ctx context.Context, imgRef image.ImageReference) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)
	licenseFileList, err := p.getDataToValidate(ctx, imgRef.ImageFSPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, errLicensesNotADir) {
			logger.Info(fmt.Sprintf("warning: licenses directory does not exist or all of its children are empty directories: %s", err))
			return false, nil
		}
		//coverage:ignore
		return false, fmt.Errorf("could not get license file list: %v", err)
	}
	return p.validate(ctx, licenseFileList)
}

//nolint:unparam // ctx is unused. Keep for future use.
func (p *HasLicenseCheck) getDataToValidate(ctx context.Context, mountedPath string) ([]fs.DirEntry, error) {
	logger := logr.FromContextOrDiscard(ctx)
	mountRoot := canonicalMountRoot(mountedPath)
	fullPath := filepath.Join(mountedPath, licensePath)
	fileinfo, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("error when checking for %s: %w", licensePath, err)
	}
	if !fileinfo.IsDir() {
		//coverage:ignore
		return nil, fmt.Errorf("%s is not a directory: %w", licensePath, errLicensesNotADir)
	}

	// WalkDir does not follow a symlink at the root; if /licenses is a symlink to a directory,
	// Stat above follows it and IsDir is true, but the walk would otherwise treat the symlink
	// itself as a non-directory "file". Resolve before walking.
	walkRoot := fullPath
	if rp, errSy := filepath.EvalSymlinks(fullPath); errSy == nil && rp != "" && pathWithinMount(mountRoot, rp) {
		walkRoot = rp
	}

	var files []fs.DirEntry
	err = filepath.WalkDir(walkRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			//coverage:ignore
			return err
		}
		if d.IsDir() {
			return nil
		}
		resolved, errEv := filepath.EvalSymlinks(p)
		if errEv != nil || resolved == "" || !pathWithinMount(mountRoot, resolved) {
			logger.V(log.DBG).Info("skipping broken or escaped link", "path", p, "error", errEv)
			return nil
		}
		// Stat the same path we validated (resolved), not p, to avoid a symlink TOCTOU.
		fi, errSt := os.Stat(resolved)
		if errSt != nil || !fi.Mode().IsRegular() {
			return nil
		}
		files = append(files, resolvedDirEntry{DirEntry: d, fi: fi})
		return nil
	})
	if err != nil {
		//coverage:ignore
		return nil, fmt.Errorf("could not walk directory %s: %w", licensePath, err)
	}
	return files, nil
}

//nolint:unparam // ctx is unused. Keep for future use.
func (p *HasLicenseCheck) validate(ctx context.Context, licenseFileList []fs.DirEntry) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)

	nonZeroLength := slices.ContainsFunc(licenseFileList, func(f fs.DirEntry) bool {
		info, err := f.Info()
		if err != nil {
			//coverage:ignore
			return false
		}
		return info.Size() > 0
	})

	logger.V(log.DBG).Info("number of licenses found", "licenseCount", len(licenseFileList))
	return len(licenseFileList) >= minLicenseFileCount && nonZeroLength, nil
}

func (p *HasLicenseCheck) Name() string {
	return "HasLicense"
}

func (p *HasLicenseCheck) Metadata() check.Metadata {
	return check.Metadata{
		Description:      "Checking if terms and conditions applicable to the software including open source licensing information are present. The license must be at /licenses",
		Level:            "best",
		KnowledgeBaseURL: certDocumentationURL,
		CheckURL:         certDocumentationURL,
	}
}

func (p *HasLicenseCheck) Help() check.HelpText {
	return check.HelpText{
		Message:    "Check HasLicense encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Create a directory named /licenses and include all relevant licensing and/or terms and conditions as text file(s) in that directory.",
	}
}

func (p *HasLicenseCheck) RequiredFilePatterns() []string {
	//coverage:ignore
	return []string{filepath.Join(licensePath, "**")}
}
