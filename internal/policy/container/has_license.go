package container

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"

	"github.com/go-logr/logr"
)

const (
	licensePath         = "/licenses"
	minLicenseFileCount = 1
)

var errLicensesNotADir = errors.New("licenses is not a directory")

var _ check.Check = &HasLicenseCheck{}

// HasLicenseCheck evaluates that the image contains a license definition available at
// /licenses.
type HasLicenseCheck struct{}

func (p *HasLicenseCheck) Validate(ctx context.Context, imgRef image.ImageReference) (bool, error) {
	licenseFileList, err := p.getDataToValidate(ctx, imgRef.ImageFSPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, errLicensesNotADir) {
			return false, nil
		}
		return false, fmt.Errorf("could not get license file list: %v", err)
	}
	return p.validate(ctx, licenseFileList)
}

//nolint:unparam // ctx is unused. Keep for future use.
func (p *HasLicenseCheck) getDataToValidate(ctx context.Context, mountedPath string) ([]fs.DirEntry, error) {
	fullPath := filepath.Join(mountedPath, licensePath)
	fileinfo, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("error when checking for %s: %w", licensePath, err)
	}
	if !fileinfo.IsDir() {
		return nil, fmt.Errorf("%s is not a directory: %w", licensePath, errLicensesNotADir)
	}

	files, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("could not read directory %s: %w", licensePath, err)
	}
	return files, nil
}

//nolint:unparam // ctx is unused. Keep for future use.
func (p *HasLicenseCheck) validate(ctx context.Context, licenseFileList []fs.DirEntry) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)

	nonZeroLength := false
	for _, f := range licenseFileList {
		info, err := f.Info()
		if err != nil {
			continue
		}
		if info.Size() > 0 {
			nonZeroLength = true
			break
		}
	}
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
	return []string{filepath.Join(licensePath, "*")}
}
