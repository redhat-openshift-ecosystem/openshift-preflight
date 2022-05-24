package container

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

const (
	licensePath         = "/licenses"
	newLine             = "\n"
	minLicenseFileCount = 1
)

// HasLicenseCheck evaluates that the image contains a license definition available at
// /licenses.
type HasLicenseCheck struct{}

func (p *HasLicenseCheck) Validate(ctx context.Context, imgRef certification.ImageReference) (bool, error) {
	licenseFileList, err := p.getDataToValidate(ctx, imgRef.ImageFSPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, ErrLicensesNotADir) {
			return false, nil
		}
		return false, err
	}
	return p.validate(ctx, licenseFileList)
}

func (p *HasLicenseCheck) getDataToValidate(ctx context.Context, mountedPath string) ([]fs.DirEntry, error) {
	fullPath := filepath.Join(mountedPath, licensePath)
	fileinfo, err := os.Stat(fullPath)
	if err != nil {
		log.Errorf("Error when checking for %s : %s", licensePath, err)
		return nil, err
	}
	if !fileinfo.IsDir() {
		log.Errorf("%s is not a directory", licensePath)
		return nil, ErrLicensesNotADir
	}

	files, err := os.ReadDir(fullPath)
	if err != nil {
		log.Errorf("Error when reading directory %s: %s", licensePath, err)
		return nil, err
	}
	return files, nil
}

func (p *HasLicenseCheck) validate(ctx context.Context, licenseFileList []fs.DirEntry) (bool, error) {
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
	log.Debugf("%d Licenses found", len(licenseFileList))
	return len(licenseFileList) >= minLicenseFileCount && nonZeroLength, nil
}

func (p *HasLicenseCheck) Name() string {
	return "HasLicense"
}

func (p *HasLicenseCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if terms and conditions applicable to the software including open source licensing information are present. The license must be at /licenses",
		Level:            "best",
		KnowledgeBaseURL: certDocumentationURL,
		CheckURL:         certDocumentationURL,
	}
}

func (p *HasLicenseCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check HasLicense encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Create a directory named /licenses and include all relevant licensing and/or terms and conditions as text file(s) in that directory.",
	}
}
