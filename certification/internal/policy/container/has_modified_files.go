package container

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/rpm"
	log "github.com/sirupsen/logrus"
)

// HasModifiedFilesCheck evaluates that no files from the base layer have been modified by
// subsequent layers by comparing the file list installed by Packages against the file list
// modified in subsequent layers.
type HasModifiedFilesCheck struct{}

type packageFilesRef struct {
	// LayerFiles contains a slice of files modified by layer.
	LayerFiles [][]string
	// PackagesFiles is a map of modified files. the anonymous struct
	// here is just used to allow us to search package files by name
	// as the key of this map, and doesn't have any value.
	PackageFiles map[string]struct{}
}

func (p *HasModifiedFilesCheck) Validate(ctx context.Context, imgRef certification.ImageReference) (bool, error) {
	packageFiles, err := p.getDataToValidate(ctx, imgRef)
	if err != nil {
		return false, fmt.Errorf("%w: %s", ErrDetectingModifiedFiles, err)
	}
	return p.validate(packageFiles)
}

// getDataToValidate returns the list of files (packageFilesRef.PackageFiles)
// installed via packages from the container image,and the list of files (packageFilesRef.LayerFiles)
// modified/added via layers in the image.
func (p *HasModifiedFilesCheck) getDataToValidate(ctx context.Context, imgRef certification.ImageReference) (*packageFilesRef, error) {
	// Get a list of packages from the RPM database. This avoids having to rely on
	// rpm, dnf, yum, etc. being installed in the image.
	pkgList, err := rpm.GetPackageList(ctx, imgRef.ImageFSPath)
	if err != nil {
		return nil, fmt.Errorf("could not get rpm list: %w", err)
	}

	// Get the files put in place on the filesystem by the
	// installed packages.
	packageFiles := make(map[string]struct{}, len(pkgList))
	for _, pkg := range pkgList {
		filenames, err := pkg.InstalledFiles()
		if err != nil {
			return nil, err
		}
		for _, file := range filenames {
			// A struct is used here, but it is unimportant and
			// should not have value.
			packageFiles[file] = struct{}{}
		}
	}

	layers, err := imgRef.ImageInfo.Layers()
	if err != nil {
		return nil, err
	}

	files := make([][]string, 0, len(layers))

	// Uncompress each layer and build a slice containing the files
	// modified by each layer.
	for _, layer := range layers {
		r, err := layer.Uncompressed()
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrExtractingLayer, err)
		}
		pathChan := make(chan string)

		go func() {
			// For each path in the pathChan, add it to the layer's
			// list of files.
			layerFiles := make([]string, 0)
			for path := range pathChan {
				layerFiles = append(layerFiles, path)
			}
			// also add it to the overall list of files.
			files = append(files, layerFiles)
		}()
		// add paths to the pathChan
		err = untar(pathChan, r)
		if err != nil {
			return nil, fmt.Errorf("failed to extract tarball: %w", err)
		}
	}

	// In cases where the image was built FROM scratch,
	// we drop the empty layer. This should be relatively rare,
	// but it is the case in images such as ubi-micro. This works
	// around using that empty layer as a base layer in Validate().
	if len(files[0]) == 0 {
		diff0, _ := layers[0].DiffID()
		diff1, _ := layers[1].DiffID()
		log.Debugf(
			"The first layer (%s) contained no files, so the next layer (%s) is being used as the base layer.",
			diff0.String(),
			diff1.String(),
		)
		files = files[1:] // shift the empty layer out.
	}

	return &packageFilesRef{files, packageFiles}, nil
}

// validate compares the list of LayerFiles and PackageFiles to see what PackageFiles
// have been modified within the additional layers.
func (p *HasModifiedFilesCheck) validate(packageFilesRef *packageFilesRef) (bool, error) {
	layerFiles := packageFilesRef.LayerFiles
	packageFiles := packageFilesRef.PackageFiles

	// Determine the list of files that were a part of the base layer.
	baseLayer := make(map[string]struct{}, len(layerFiles[0]))
	for _, filename := range layerFiles[0] {
		if _, ok := packageFiles[filename]; ok {
			baseLayer[filename] = struct{}{}
		}
	}
	layers := layerFiles[1:]

	// Look for modifications in subsequent layers by determining
	// if a file in a base layer exists in a subsequent layer.
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
		KnowledgeBaseURL: certDocumentationURL,
		CheckURL:         certDocumentationURL,
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
