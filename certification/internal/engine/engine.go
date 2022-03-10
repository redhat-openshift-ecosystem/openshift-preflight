package engine

import (
	"archive/tar"
	"bytes"
	"crypto/md5"
	"encoding/json"
	syserrors "errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/pyxis"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	log "github.com/sirupsen/logrus"
)

// CraneEngine implements a certification.CheckEngine, and leverage crane to interact with
// the container registry and target image.
type CraneEngine struct {
	// Image is what is being tested, and should contain the
	// fully addressable path (including registry, namespaces, etc)
	// to the image
	Image string
	// Checks is an array of all checks to be executed against
	// the image provided.
	Checks []certification.Check

	// IsBundle is an indicator that the asset is a bundle.
	IsBundle bool

	// IsScratch is an indicator that the asset is a scratch image
	IsScratch bool

	imageRef certification.ImageReference
	results  runtime.Results
}

func (c *CraneEngine) ExecuteChecks() error {
	log.Debug("target image: ", c.Image)

	// prepare crane runtime options, if necessary
	options := make([]crane.Option, 0)

	// pull the image and save to fs
	log.Debug("pulling image from target registry")
	img, err := crane.Pull(c.Image, options...)
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrGetRemoteContainerFailed, err)
	}

	// create tmpdir to receive extracted fs
	tmpdir, err := os.MkdirTemp(os.TempDir(), "preflight-*")
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrCreateTempDir, err)
	}
	log.Debug("temporary directory is ", tmpdir)
	defer func() {
		if err := os.RemoveAll(tmpdir); err != nil {
			log.Error("unable to clean up tmpdir", tmpdir, err)
		}
	}()

	containerFSPath := path.Join(tmpdir, "fs")
	if err := os.Mkdir(containerFSPath, 0o755); err != nil {
		return fmt.Errorf("%w: %s: %s", errors.ErrCreateTempDir, containerFSPath, err)
	}

	// export/flatten, and extract
	log.Debug("exporting and flattening image")
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		log.Debugf("writing container filesystem to output dir: %s", containerFSPath)
		err = crane.Export(img, w)
		if err != nil {
			// TODO: Handle this error more effectively. Right now we rely on
			// error handling in the logic to extract this export in a lower
			// line, but we should probably exit early if the export encounters
			// an error, which requires watching multiple error streams.
			log.Error("unable to export and flatten container filesystem:", err)
		}
	}()

	log.Debug("extracting container filesystem to ", containerFSPath)
	if err := untar(containerFSPath, r); err != nil {
		return fmt.Errorf("%w: %s", errors.ErrExtractingTarball, err)
	}

	reference, err := name.ParseReference(c.Image)
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrInvalidImageUri, err)
	}

	// store the image internals in the engine image reference to pass to validations.
	c.imageRef = certification.ImageReference{
		ImageURI:        c.Image,
		ImageFSPath:     containerFSPath,
		ImageInfo:       img,
		ImageRegistry:   reference.Context().RegistryStr(),
		ImageRepository: reference.Context().RepositoryStr(),
		ImageTagOrSha:   reference.Identifier(),
	}

	if err := writeCertImage(c.imageRef); err != nil {
		return err
	}

	if !c.IsScratch {
		if err := writeRPMManifest(containerFSPath); err != nil {
			return err
		}
	}

	if c.IsBundle {
		// Record test cluster version
		c.results.TestedOn, err = GetOpenshiftClusterVersion()
		if err != nil {
			log.Error("Unable to determine test cluster version: ", err)
		}
	} else {
		log.Debug("Container checks do not require a cluster. skipping cluster version check.")
		c.results.TestedOn = runtime.UnknownOpenshiftClusterVersion()
	}

	// execute checks
	log.Debug("executing checks")
	for _, check := range c.Checks {
		c.results.TestedImage = c.Image

		log.Debug("running check: ", check.Name())

		// run the validation
		checkStartTime := time.Now()
		checkPassed, err := check.Validate(c.imageRef)
		checkElapsedTime := time.Since(checkStartTime)

		if err != nil {
			log.WithFields(log.Fields{"result": err, "ERROR": err.Error()}).Info("check completed: ", check.Name())
			c.results.Errors = append(c.results.Errors, runtime.Result{Check: check, ElapsedTime: checkElapsedTime})
			continue
		}

		if !checkPassed {
			log.WithFields(log.Fields{"result": "FAILED"}).Info("check completed: ", check.Name())
			c.results.Failed = append(c.results.Failed, runtime.Result{Check: check, ElapsedTime: checkElapsedTime})
			continue
		}

		log.WithFields(log.Fields{"result": "PASSED"}).Info("check completed: ", check.Name())
		c.results.Passed = append(c.results.Passed, runtime.Result{Check: check, ElapsedTime: checkElapsedTime})
	}

	if len(c.results.Errors) > 0 || len(c.results.Failed) > 0 {
		c.results.PassedOverall = false
	} else {
		c.results.PassedOverall = true
	}

	// hash contents if bundle
	if c.IsBundle {
		md5sum, err := generateBundleHash(c.imageRef.ImageFSPath)
		if err != nil {
			log.Debugf("could not generate bundle hash")
		}
		c.results.CertificationHash = md5sum
	}

	return nil
}

func generateBundleHash(bundlePath string) (string, error) {
	files := make(map[string]string)
	fileSystem := os.DirFS(bundlePath)

	hashBuffer := bytes.Buffer{}

	fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Errorf("could not read bundle directory: %s", path)
			return err
		}
		if d.Name() == "Dockerfile" {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		filebytes, err := fs.ReadFile(fileSystem, path)
		if err != nil {
			log.Errorf("could not read file: %s", path)
			return err
		}
		md5sum := fmt.Sprintf("%x", md5.Sum(filebytes))
		files[md5sum] = fmt.Sprintf("./%s", path)
		return nil
	})

	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	for _, k := range keys {
		hashBuffer.WriteString(fmt.Sprintf("%s  %s\n", k, files[k]))
	}

	_, err := artifacts.WriteFile("hashes.txt", hashBuffer.String())
	if err != nil {
		return "", err
	}

	sum := fmt.Sprintf("%x", md5.Sum(hashBuffer.Bytes()))

	log.Debugf("md5 sum: %s", sum)

	return sum, nil
}

// Results will return the results of check execution.
func (c *CraneEngine) Results() runtime.Results {
	return c.results
}

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func untar(dst string, r io.Reader) error {
	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0o755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()

			// if it's a link create it
		case tar.TypeSymlink:
			err := os.Symlink(header.Linkname, filepath.Join(dst, header.Name))
			if err != nil {
				log.Println(fmt.Sprintf("Error creating link: %s. Ignoring.", header.Name))
				continue
			}
		}
	}
}

func writeCertImage(imageRef certification.ImageReference) error {
	config, err := imageRef.ImageInfo.ConfigFile()
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrImageInspectFailed, err)
	}

	manifest, err := imageRef.ImageInfo.Manifest()
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrImageInspectFailed, err)
	}

	digest, err := imageRef.ImageInfo.Digest()
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrImageInspectFailed, err)
	}

	rawConfig, err := imageRef.ImageInfo.RawConfigFile()
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrImageInspectFailed, err)
	}

	size, err := imageRef.ImageInfo.Size()
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrImageInspectFailed, err)
	}

	labels := convertLabels(config.Config.Labels)
	layerSizes := make([]pyxis.Layer, 0, len(config.RootFS.DiffIDs))
	for _, diffid := range config.RootFS.DiffIDs {
		layer, err := imageRef.ImageInfo.LayerByDiffID(diffid)
		if err != nil {
			return err
		}

		uncompressed, err := layer.Uncompressed()
		if err != nil {
			return err
		}
		written, err := io.Copy(io.Discard, uncompressed)
		if err != nil {
			return err
		}

		pyxisLayer := pyxis.Layer{
			LayerId: diffid.String(),
			Size:    written,
		}
		layerSizes = append(layerSizes, pyxisLayer)
	}

	sumLayersSizeBytes, err := sumLayerSizeBytes(layerSizes)
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrImageInspectFailed, err)
	}

	addedDate := time.Now().UTC().Format(time.RFC3339)

	log.Debug("getting tag info for image")
	tag, err := name.NewTag(imageRef.ImageURI)
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrParseTagInfoFailed, err)
	}

	tags := make([]pyxis.Tag, 0, 1)
	tags = append(tags, pyxis.Tag{
		AddedDate: addedDate,
		Name:      tag.TagStr(),
	})

	repositories := make([]pyxis.Repository, 0, 1)
	repositories = append(repositories, pyxis.Repository{
		PushDate:   addedDate,
		Registry:   tag.Context().RegistryStr(),
		Repository: tag.Context().RepositoryStr(),
		Tags:       tags,
	})

	certImage := pyxis.CertImage{
		DockerImageDigest: digest.String(),
		DockerImageID:     manifest.Config.Digest.String(),
		ImageID:           digest.String(),
		Architecture:      config.Architecture,
		ParsedData: &pyxis.ParsedData{
			Architecture:           config.Architecture,
			Command:                strings.Join(config.Config.Cmd, " "),
			Created:                config.Created.String(),
			DockerVersion:          config.DockerVersion,
			ImageID:                digest.String(),
			Labels:                 labels,
			OS:                     config.OS,
			Size:                   size,
			UncompressedLayerSizes: layerSizes,
		},
		RawConfig:         string(rawConfig),
		Repositories:      repositories,
		SumLayerSizeBytes: sumLayersSizeBytes,
		// This is an assumption that the DiffIDs are in order from base up.
		// Need more evisdence that this is always the case.
		UncompressedTopLayerId: config.RootFS.DiffIDs[0].String(),
	}

	// calling MarshalIndent so the json file written to disk is human-readable when opened
	certImageJson, err := json.MarshalIndent(certImage, "", "    ")
	if err != nil {
		return err
	}

	fileName, err := artifacts.WriteFile(certification.DefaultCertImageFilename, string(certImageJson))
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrSaveFileFailed, err)
	}

	log.Tracef("image config written to disk: %s", fileName)

	return nil
}

func writeRPMManifest(containerFSPath string) error {
	// Check for rpmdb.sqlite. If not found, check for Packages
	rpmdirPath := filepath.Join(containerFSPath, "var", "lib", "rpm")
	rpmdbPath := filepath.Join(rpmdirPath, "rpmdb.sqlite")

	if _, err := os.Stat(rpmdbPath); syserrors.Is(err, os.ErrNotExist) {
		// rpmdb.sqlite doesn't exist. Fall back to Packages
		rpmdbPath = filepath.Join(rpmdirPath, "Packages")
	}

	db, err := rpmdb.Open(rpmdbPath)
	if err != nil {
		return err
	}

	pkgList, err := db.ListPackages()
	if err != nil {
		return err
	}

	// covert rpm struct to pxyis struct
	rpms := make([]pyxis.RPM, 0, len(pkgList))
	for _, packageInfo := range pkgList {

		var bgName, endChop, srpmNevra, pgpKeyId string

		// accounting for the fact that not all packages have a source rpm
		if len(packageInfo.SourceRpm) > 0 {
			bgName = packageInfo.SourceRpm[0:strings.LastIndex(strings.SplitAfter(packageInfo.SourceRpm, ".")[0], "-")]
			endChop = strings.TrimPrefix(strings.TrimSuffix(regexp.MustCompile("(-[0-9].*)").FindString(packageInfo.SourceRpm), ".rpm"), "-")

			srpmNevra = fmt.Sprintf("%s-%d:%s", bgName, packageInfo.Epoch, endChop)
		}

		if len(packageInfo.PGP) > 0 {
			matches := regexp.MustCompile(".*, Key ID (.*)").FindStringSubmatch(packageInfo.PGP)
			if matches != nil {
				pgpKeyId = matches[1]
			} else {
				log.Debugf("string did not match the format required: %s", packageInfo.PGP)
				pgpKeyId = ""
			}
		}

		rpm := pyxis.RPM{
			Architecture: packageInfo.Arch,
			Gpg:          pgpKeyId,
			Name:         packageInfo.Name,
			Nvra:         fmt.Sprintf("%s-%s-%s.%s", packageInfo.Name, packageInfo.Version, packageInfo.Release, packageInfo.Arch),
			Release:      packageInfo.Release,
			SrpmName:     bgName,
			SrpmNevra:    srpmNevra,
			Summary:      packageInfo.Summary,
			Version:      packageInfo.Version,
		}

		rpms = append(rpms, rpm)
	}

	rpmManifest := pyxis.RPMManifest{
		RPMS: rpms,
	}

	// calling MarshalIndent so the json file written to disk is human-readable when opened
	rpmManifestJson, err := json.MarshalIndent(rpmManifest, "", "    ")
	if err != nil {
		return err
	}

	fileName, err := artifacts.WriteFile(certification.DefaultRPMManifestFilename, string(rpmManifestJson))
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrSaveFileFailed, err)
	}

	log.Tracef("rpm manifest written to disk: %s", fileName)

	return nil
}

func sumLayerSizeBytes(layers []pyxis.Layer) (int64, error) {
	var sum int64
	for _, layer := range layers {
		sum += layer.Size
	}

	return sum, nil
}

func convertLabels(imageLabels map[string]string) []pyxis.Label {
	pyxisLabels := make([]pyxis.Label, 0, len(imageLabels))
	for key, value := range imageLabels {
		label := pyxis.Label{
			Name:  key,
			Value: value,
		}

		pyxisLabels = append(pyxisLabels, label)
	}

	return pyxisLabels
}
