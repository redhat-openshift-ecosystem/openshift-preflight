package engine

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/authn"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/openshift"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/rpm"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/pyxis"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	log "github.com/sirupsen/logrus"
)

// CraneEngine implements a certification.CheckEngine, and leverage crane to interact with
// the container registry and target image.
type CraneEngine struct {
	Config certification.Config
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

func (c *CraneEngine) ExecuteChecks(ctx context.Context) error {
	log.Debug("target image: ", c.Image)

	if c.Config == nil {
		return fmt.Errorf("a runtime configuration was not provided")
	}

	// prepare crane runtime options, if necessary
	options := []crane.Option{
		crane.WithContext(ctx),
		crane.WithAuthFromKeychain(
			authn.PreflightKeychain(
				// We configure the Preflight Keychain here.
				// In theory, we should not require further configuration
				// downstream because the PreflightKeychain is a singleton.
				// However, as long as we pass this same DockerConfig
				// value downstream, it shouldn't matter if the
				// keychain is reconfigured downstream.
				authn.WithDockerConfig(c.Config.DockerConfig()),
			),
		),
	}

	// pull the image and save to fs
	log.Debug("pulling image from target registry")
	img, err := crane.Pull(c.Image, options...)
	if err != nil {
		return fmt.Errorf("failed to pull remote container: %v", err)
	}

	// create tmpdir to receive extracted fs
	tmpdir, err := os.MkdirTemp(os.TempDir(), "preflight-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}
	log.Debug("temporary directory is ", tmpdir)
	defer func() {
		if err := os.RemoveAll(tmpdir); err != nil {
			log.Errorf("unable to clean up tmpdir %s: %v", tmpdir, err)
		}
	}()

	imageTarPath := path.Join(tmpdir, "cache")
	if err := os.Mkdir(imageTarPath, 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %s: %v", imageTarPath, err)
	}

	img = cache.Image(img, cache.NewFilesystemCache(imageTarPath))

	containerFSPath := path.Join(tmpdir, "fs")
	if err := os.Mkdir(containerFSPath, 0o755); err != nil {
		return fmt.Errorf("failed to create container expansion directory: %s: %v", containerFSPath, err)
	}

	// export/flatten, and extract
	log.Debug("exporting and flattening image")
	wg := sync.WaitGroup{}
	wg.Add(1)
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
		wg.Done()
	}()

	log.Debug("extracting container filesystem to ", containerFSPath)
	if err := untar(containerFSPath, r); err != nil {
		return fmt.Errorf("failed to extract tarball: %v", err)
	}
	wg.Wait()

	reference, err := name.ParseReference(c.Image)
	if err != nil {
		return fmt.Errorf("image uri could not be parsed: %v", err)
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

	if err := writeCertImage(ctx, c.imageRef); err != nil {
		return fmt.Errorf("could not write cert image: %v", err)
	}

	if !c.IsScratch {
		if err := writeRPMManifest(ctx, containerFSPath); err != nil {
			return fmt.Errorf("could not write rpm manifest: %v", err)
		}
	}

	if c.IsBundle {
		// Record test cluster version
		c.results.TestedOn, err = openshift.GetOpenshiftClusterVersion()
		if err != nil {
			log.Errorf("could not determine test cluster version: %v", err)
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
		if check.Metadata().Level == "optional" {
			log.Infof("Check %s is not currently being enforced.", check.Name())
		}

		// run the validation
		checkStartTime := time.Now()
		checkPassed, err := check.Validate(ctx, c.imageRef)
		checkElapsedTime := time.Since(checkStartTime)

		if err != nil {
			log.WithFields(log.Fields{"result": "ERROR", "err": err}).Info("check completed: ", check.Name())
			c.results.Errors = appendUnlessOptional(c.results.Errors, runtime.Result{Check: check, ElapsedTime: checkElapsedTime})
			continue
		}

		if !checkPassed {
			log.WithFields(log.Fields{"result": "FAILED"}).Info("check completed: ", check.Name())
			c.results.Failed = appendUnlessOptional(c.results.Failed, runtime.Result{Check: check, ElapsedTime: checkElapsedTime})
			continue
		}

		log.WithFields(log.Fields{"result": "PASSED"}).Info("check completed: ", check.Name())
		c.results.Passed = appendUnlessOptional(c.results.Passed, runtime.Result{Check: check, ElapsedTime: checkElapsedTime})
	}

	if len(c.results.Errors) > 0 || len(c.results.Failed) > 0 {
		c.results.PassedOverall = false
	} else {
		c.results.PassedOverall = true
	}

	if c.IsBundle { // for operators:
		// hash the contents of the bundle.
		md5sum, err := generateBundleHash(c.imageRef.ImageFSPath)
		if err != nil {
			log.Errorf("could not generate bundle hash: %v", err)
		}
		c.results.CertificationHash = md5sum
	} else { // for containers:
		// Inform the user about the sha/tag binding.

		// By this point, we should have already resolved the digest so
		// we don't handle this error, but fail safe and don't log a potentially
		// incorrect line message to the user.
		if resolvedDigest, err := c.imageRef.ImageInfo.Digest(); err == nil {
			msg, logfunc := tagDigestBindingInfo(c.imageRef.ImageTagOrSha, resolvedDigest.String())
			logfunc(msg)
		}
	}

	return nil
}

func appendUnlessOptional(results []runtime.Result, result runtime.Result) []runtime.Result {
	if result.Check.Metadata().Level == "optional" {
		return results
	}
	return append(results, result)
}

// tagDigestBindingInfo emits a log line describing tag and digest binding semantics.
// The providedIdentifer is the tag or digest of the image as the user gave it at the commandline.
// resolvedDigest
func tagDigestBindingInfo(providedIdentifier string, resolvedDigest string) (msg string, logFunc func(...interface{})) {
	if strings.HasPrefix(providedIdentifier, "sha256:") {
		return "You've provided an image by digest. " +
				"When submitting this image to Red Hat for certification, " +
				"no tag will be associated with this image. " +
				"If you would like to associate a tag with this image, " +
				"please rerun this tool replacing your image reference with a tag.",
			log.Warn
	}

	return fmt.Sprintf(
		`This image's tag %s will be paired with digest %s`+
			`once this image has been published in accordance `+
			`with Red Hat Certification policy. `+
			`You may then add or remove any supplemental tags `+
			`through your Red Hat Connect portal as you see fit.`,
		providedIdentifier, resolvedDigest,
	), log.Info
}

func generateBundleHash(bundlePath string) (string, error) {
	files := make(map[string]string)
	fileSystem := os.DirFS(bundlePath)

	hashBuffer := bytes.Buffer{}

	_ = fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("could not read bundle directory: %s: %w", path, err)
		}
		if d.Name() == "Dockerfile" {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		filebytes, err := fs.ReadFile(fileSystem, path)
		if err != nil {
			return fmt.Errorf("could not read file: %s: %w", path, err)
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

	_, err := artifacts.WriteFile("hashes.txt", &hashBuffer)
	if err != nil {
		return "", fmt.Errorf("could not write hash file to artifacts dir: %w", err)
	}

	sum := fmt.Sprintf("%x", md5.Sum(hashBuffer.Bytes()))

	log.Debugf("md5 sum: %s", sum)

	return sum, nil
}

// Results will return the results of check execution.
func (c *CraneEngine) Results(ctx context.Context) runtime.Results {
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

// writeCertImage takes imageRef and writes it to disk as JSON representing a pyxis.CertImage
// struct. The file is written at path certification.DefaultCertImageFilename.
//
//nolint:unparam // ctx is unused. Keep for future use.
func writeCertImage(ctx context.Context, imageRef certification.ImageReference) error {
	config, err := imageRef.ImageInfo.ConfigFile()
	if err != nil {
		return fmt.Errorf("failed to get image config file: %w", err)
	}

	manifest, err := imageRef.ImageInfo.Manifest()
	if err != nil {
		return fmt.Errorf("failed to get image manifest: %w", err)
	}

	digest, err := imageRef.ImageInfo.Digest()
	if err != nil {
		return fmt.Errorf("failed to get image digest: %w", err)
	}

	rawConfig, err := imageRef.ImageInfo.RawConfigFile()
	if err != nil {
		return fmt.Errorf("failed to image raw config file: %w", err)
	}

	size, err := imageRef.ImageInfo.Size()
	if err != nil {
		return fmt.Errorf("failed to get image size: %w", err)
	}

	labels := convertLabels(config.Config.Labels)
	layerSizes := make([]pyxis.Layer, 0, len(config.RootFS.DiffIDs))
	for _, diffid := range config.RootFS.DiffIDs {
		layer, err := imageRef.ImageInfo.LayerByDiffID(diffid)
		if err != nil {
			return fmt.Errorf("could not get layer by diff id: %w", err)
		}

		uncompressed, err := layer.Uncompressed()
		if err != nil {
			return fmt.Errorf("could not get uncompressed layer: %w", err)
		}
		written, err := io.Copy(io.Discard, uncompressed)
		if err != nil {
			return fmt.Errorf("could not copy from layer: %w", err)
		}

		pyxisLayer := pyxis.Layer{
			LayerID: diffid.String(),
			Size:    written,
		}
		layerSizes = append(layerSizes, pyxisLayer)
	}

	manifestLayers := make([]string, 0, len(manifest.Layers))
	for _, layer := range manifest.Layers {
		manifestLayers = append(manifestLayers, layer.Digest.String())
	}

	sumLayersSizeBytes := sumLayerSizeBytes(layerSizes)

	addedDate := time.Now().UTC().Format(time.RFC3339)

	tags := make([]pyxis.Tag, 0, 1)
	tags = append(tags, pyxis.Tag{
		AddedDate: addedDate,
		Name:      imageRef.ImageTagOrSha,
	})

	repositories := make([]pyxis.Repository, 0, 1)
	repositories = append(repositories, pyxis.Repository{
		PushDate:   addedDate,
		Registry:   imageRef.ImageRegistry,
		Repository: imageRef.ImageRepository,
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
			Layers:                 manifestLayers,
			OS:                     config.OS,
			Size:                   size,
			UncompressedLayerSizes: layerSizes,
		},
		RawConfig:         string(rawConfig),
		Repositories:      repositories,
		SumLayerSizeBytes: sumLayersSizeBytes,
		// This is an assumption that the DiffIDs are in order from base up.
		// Need more evidence that this is always the case.
		UncompressedTopLayerID: config.RootFS.DiffIDs[0].String(),
	}

	// calling MarshalIndent so the json file written to disk is human-readable when opened
	certImageJSON, err := json.MarshalIndent(certImage, "", "    ")
	if err != nil {
		return fmt.Errorf("could not marshal cert image: %w", err)
	}

	fileName, err := artifacts.WriteFile(certification.DefaultCertImageFilename, bytes.NewReader(certImageJSON))
	if err != nil {
		return fmt.Errorf("failed to save file to artifacts directory: %w", err)
	}

	log.Tracef("image config written to disk: %s", fileName)

	return nil
}

func getBgName(srcrpm string) string {
	parts := strings.Split(srcrpm, "-")
	return strings.Join(parts[0:len(parts)-2], "-")
}

func writeRPMManifest(ctx context.Context, containerFSPath string) error {
	pkgList, err := rpm.GetPackageList(ctx, containerFSPath)
	if err != nil {
		log.Errorf("could not get rpm list, continuing without it: %v", err)
	}

	// covert rpm struct to pxyis struct
	rpms := make([]pyxis.RPM, 0, len(pkgList))
	for _, packageInfo := range pkgList {
		var bgName, endChop, srpmNevra, pgpKeyID string

		// accounting for the fact that not all packages have a source rpm
		if len(packageInfo.SourceRpm) > 0 {
			bgName = getBgName(packageInfo.SourceRpm)
			endChop = strings.TrimPrefix(strings.TrimSuffix(regexp.MustCompile("(-[0-9].*)").FindString(packageInfo.SourceRpm), ".rpm"), "-")

			srpmNevra = fmt.Sprintf("%s-%d:%s", bgName, packageInfo.Epoch, endChop)
		}

		if len(packageInfo.PGP) > 0 {
			matches := regexp.MustCompile(".*, Key ID (.*)").FindStringSubmatch(packageInfo.PGP)
			if matches != nil {
				pgpKeyID = matches[1]
			} else {
				log.Debugf("string did not match the format required: %s", packageInfo.PGP)
				pgpKeyID = ""
			}
		}

		pyxisRPM := pyxis.RPM{
			Architecture: packageInfo.Arch,
			Gpg:          pgpKeyID,
			Name:         packageInfo.Name,
			Nvra:         fmt.Sprintf("%s-%s-%s.%s", packageInfo.Name, packageInfo.Version, packageInfo.Release, packageInfo.Arch),
			Release:      packageInfo.Release,
			SrpmName:     bgName,
			SrpmNevra:    srpmNevra,
			Summary:      packageInfo.Summary,
			Version:      packageInfo.Version,
		}

		rpms = append(rpms, pyxisRPM)
	}

	rpmManifest := pyxis.RPMManifest{
		RPMS: rpms,
	}

	// calling MarshalIndent so the json file written to disk is human-readable when opened
	rpmManifestJSON, err := json.MarshalIndent(rpmManifest, "", "    ")
	if err != nil {
		return fmt.Errorf("could not marshal rpm manifest: %w", err)
	}

	fileName, err := artifacts.WriteFile(certification.DefaultRPMManifestFilename, bytes.NewReader(rpmManifestJSON))
	if err != nil {
		return fmt.Errorf("failed to save file to artifacts directory: %w", err)
	}

	log.Tracef("rpm manifest written to disk: %s", fileName)

	return nil
}

func sumLayerSizeBytes(layers []pyxis.Layer) int64 {
	var sum int64
	for _, layer := range layers {
		sum += layer.Size
	}

	return sum
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
