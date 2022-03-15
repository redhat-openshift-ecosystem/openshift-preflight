package container

import (
	"archive/tar"
	"fmt"
	"io"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/rpm"
	log "github.com/sirupsen/logrus"
)

// HasLicenseCheck evaluates that the image contains a license definition available at
// /licenses.
type HasModifiedFilesCheck struct{}

type packageFilesRef struct {
	LayerFiles   [][]string
	PackageFiles map[string]struct{}
}

func (p *HasModifiedFilesCheck) Validate(imgRef certification.ImageReference) (bool, error) {
	packageFiles, err := p.getDataToValidate(imgRef)
	if err != nil {
		return false, fmt.Errorf("%w: %s", errors.ErrExtractingTarball, err)
	}
	return p.validate(packageFiles)
}

func (p *HasModifiedFilesCheck) getDataToValidate(imageRef certification.ImageReference) (*packageFilesRef, error) {
	pkgList, err := rpm.GetPackageList(imageRef.ImageFSPath)
	if err != nil {
		return nil, err
	}
	packageFiles := make(map[string]struct{}, len(pkgList))
	for _, pkg := range pkgList {
		filenames, err := pkg.InstalledFiles()
		if err != nil {
			return nil, err
		}
		for _, file := range filenames {
			packageFiles[file] = struct{}{}
		}
	}

	layers, err := imageRef.ImageInfo.Layers()
	if err != nil {
		return nil, err
	}

	files := make([][]string, 0, len(layers))

	for _, layer := range layers {
		r, err := layer.Uncompressed()
		if err != nil {
			return nil, fmt.Errorf("%w: %s", errors.ErrExtractingTarball, err)
		}
		pathChan := make(chan string)

		go func() {
			layerFiles := make([]string, 0)
			for path := range pathChan {
				layerFiles = append(layerFiles, path)
			}
			files = append(files, layerFiles)
		}()
		err = untar(pathChan, r)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", errors.ErrExtractingTarball, err)
		}
	}

	return &packageFilesRef{files, packageFiles}, nil
}

func (p *HasModifiedFilesCheck) validate(packageFilesRef *packageFilesRef) (bool, error) {
	layerFiles := packageFilesRef.LayerFiles
	packageFiles := packageFilesRef.PackageFiles
	baseLayer := make(map[string]struct{}, len(layerFiles[0]))
	for _, filename := range layerFiles[0] {
		if _, ok := packageFiles[filename]; ok {
			baseLayer[filename] = struct{}{}
		}
	}
	layers := layerFiles[1:]

	modifiedFilesDetected := false
	for _, layer := range layers {
		for _, file := range layer {
			if _, ok := baseLayer[file]; ok {
				// This means the files exists in the base layer. This is a fail.
				log.Errorf("modified file detected: %s", file)
				modifiedFilesDetected = true
			}
		}
	}
	return !modifiedFilesDetected, nil
}

func (p HasModifiedFilesCheck) Name() string {
	return "HasModifiedFiles"
}

func (p HasModifiedFilesCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check HasModifiedFiles encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Do not modify any files installed by RPM in the base Red Hat layer",
	}
}

func (p HasModifiedFilesCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checks that no files installed via RPM in the base Red Hat layer have been modified",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func untar(pathChan chan<- string, r io.Reader) error {
	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			close(pathChan)
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := header.Name

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir ignore it
		case tar.TypeDir:
			continue

		// if it's a file write the name to the channel
		case tar.TypeReg:
			// Strip off any leading '/' or './'
			pathChan <- strings.TrimLeft(target, "./")

		// if it's a link create it
		case tar.TypeSymlink:
			pathChan <- strings.TrimLeft(header.Linkname, "./")
		}
	}
}
