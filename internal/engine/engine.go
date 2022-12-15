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
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/authn"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/openshift"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/operatorsdk"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy"
	containerpol "github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy/container"
	operatorpol "github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy/operator"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/pyxis"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/rpm"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/runtime"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// CraneEngine implements a certification.CheckEngine, and leverage crane to interact with
// the container registry and target image.
type CraneEngine struct {
	// Kubeconfig is a byte slice containing a valid Kubeconfig to be used by checks.
	Kubeconfig []byte
	// DockerConfig is the credential required to pull the image.
	DockerConfig string
	// Image is what is being tested, and should contain the
	// fully addressable path (including registry, namespaces, etc)
	// to the image
	Image string
	// Checks is an array of all checks to be executed against
	// the image provided.
	Checks []check.Check
	// Platform is the container platform to use. E.g. amd64.
	Platform string

	// IsBundle is an indicator that the asset is a bundle.
	IsBundle bool

	// IsScratch is an indicator that the asset is a scratch image
	IsScratch bool

	// Insecure controls whether to allow an insecure connection to
	// the registry crane connects with.
	Insecure bool

	imageRef image.ImageReference
	results  certification.Results
}

func (c *CraneEngine) ExecuteChecks(ctx context.Context) error {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(log.DBG).Info("target image", "image", c.Image)

	// prepare crane runtime options, if necessary
	options := []crane.Option{
		crane.WithContext(ctx),
		crane.WithAuthFromKeychain(
			authn.PreflightKeychain(
				ctx,
				// We configure the Preflight Keychain here.
				// In theory, we should not require further configuration
				// downstream because the PreflightKeychain is a singleton.
				// However, as long as we pass this same DockerConfig
				// value downstream, it shouldn't matter if the
				// keychain is reconfigured downstream.
				authn.WithDockerConfig(c.DockerConfig),
			),
		),
		crane.WithPlatform(&cranev1.Platform{
			OS:           "linux",
			Architecture: c.Platform,
		}),
		retryOnceAfter(5 * time.Second),
	}

	if c.Insecure {
		options = append(options, crane.Insecure)
	}

	// pull the image and save to fs
	logger.V(log.DBG).Info("pulling image from target registry")
	img, err := crane.Pull(c.Image, options...)
	if err != nil {
		return fmt.Errorf("failed to pull remote container: %v", err)
	}

	// create tmpdir to receive extracted fs
	tmpdir, err := os.MkdirTemp(os.TempDir(), "preflight-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}
	logger.V(log.DBG).Info("created temporary directory", "path", tmpdir)
	defer func() {
		if err := os.RemoveAll(tmpdir); err != nil {
			logger.Error(err, "unable to clean up tmpdir", "tempDir", tmpdir)
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
	logger.V(log.DBG).Info("exporting and flattening image")
	wg := sync.WaitGroup{}
	wg.Add(1)
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		logger.V(log.DBG).Info("writing container filesystem", "outputDirectory", containerFSPath)
		err = crane.Export(img, w)
		if err != nil {
			// TODO: Handle this error more effectively. Right now we rely on
			// error handling in the logic to extract this export in a lower
			// line, but we should probably exit early if the export encounters
			// an error, which requires watching multiple error streams.
			logger.Error(err, "unable to export and flatten container filesystem")
		}
		wg.Done()
	}()

	logger.V(log.DBG).Info("extracting container filesystem", "path", containerFSPath)
	if err := untar(ctx, containerFSPath, r); err != nil {
		return fmt.Errorf("failed to extract tarball: %v", err)
	}
	wg.Wait()

	reference, err := name.ParseReference(c.Image)
	if err != nil {
		return fmt.Errorf("image uri could not be parsed: %v", err)
	}

	// store the image internals in the engine image reference to pass to validations.
	c.imageRef = image.ImageReference{
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
		version, err := openshift.GetOpenshiftClusterVersion(ctx, c.Kubeconfig)
		if err != nil {
			logger.Error(err, "could not determine test cluster version")
		}
		c.results.TestedOn = version
	} else {
		logger.V(log.DBG).Info("Container checks do not require a cluster. skipping cluster version check.")
		c.results.TestedOn = runtime.UnknownOpenshiftClusterVersion()
	}

	// execute checks
	logger.V(log.DBG).Info("executing checks")
	for _, check := range c.Checks {
		c.results.TestedImage = c.Image

		logger.V(log.DBG).Info("running check", "check", check.Name())
		if check.Metadata().Level == "optional" {
			logger.Info(fmt.Sprintf("Check %s is not currently being enforced.", check.Name()))
		}

		// run the validation
		checkStartTime := time.Now()
		checkPassed, err := check.Validate(ctx, c.imageRef)
		checkElapsedTime := time.Since(checkStartTime)

		if err != nil {
			logger.WithValues("result", "ERROR", "err", err).Info("check completed", "check", check.Name())
			c.results.Errors = appendUnlessOptional(c.results.Errors, certification.Result{Check: check, ElapsedTime: checkElapsedTime})
			continue
		}

		if !checkPassed {
			logger.WithValues("result", "FAILED").Info("check completed", "check", check.Name())
			c.results.Failed = appendUnlessOptional(c.results.Failed, certification.Result{Check: check, ElapsedTime: checkElapsedTime})
			continue
		}

		logger.WithValues("result", "PASSED").Info("check completed", "check", check.Name())
		c.results.Passed = appendUnlessOptional(c.results.Passed, certification.Result{Check: check, ElapsedTime: checkElapsedTime})
	}

	if len(c.results.Errors) > 0 || len(c.results.Failed) > 0 {
		c.results.PassedOverall = false
	} else {
		c.results.PassedOverall = true
	}

	if c.IsBundle { // for operators:
		// hash the contents of the bundle.
		md5sum, err := generateBundleHash(ctx, c.imageRef.ImageFSPath)
		if err != nil {
			logger.Error(err, "could not generate bundle hash")
		}
		c.results.CertificationHash = md5sum
	} else { // for containers:
		// Inform the user about the sha/tag binding.

		// By this point, we should have already resolved the digest so
		// we don't handle this error, but fail safe and don't log a potentially
		// incorrect line message to the user.
		if resolvedDigest, err := c.imageRef.ImageInfo.Digest(); err == nil {
			msg, warn := tagDigestBindingInfo(c.imageRef.ImageTagOrSha, resolvedDigest.String())
			if warn {
				logger.Info(fmt.Sprintf("Warning: %s", msg))
			} else {
				logger.Info(msg)
			}
		}
	}

	return nil
}

func appendUnlessOptional(results []certification.Result, result certification.Result) []certification.Result {
	if result.Check.Metadata().Level == "optional" {
		return results
	}
	return append(results, result)
}

// tagDigestBindingInfo emits a log line describing tag and digest binding semantics.
// The providedIdentifer is the tag or digest of the image as the user gave it at the commandline.
// resolvedDigest
func tagDigestBindingInfo(providedIdentifier string, resolvedDigest string) (msg string, warn bool) {
	if strings.HasPrefix(providedIdentifier, "sha256:") {
		return "You've provided an image by digest. " +
				"When submitting this image to Red Hat for certification, " +
				"no tag will be associated with this image. " +
				"If you would like to associate a tag with this image, " +
				"please rerun this tool replacing your image reference with a tag.",
			true
	}

	return fmt.Sprintf(
		`This image's tag %s will be paired with digest %s`+
			`once this image has been published in accordance `+
			`with Red Hat Certification policy. `+
			`You may then add or remove any supplemental tags `+
			`through your Red Hat Connect portal as you see fit.`,
		providedIdentifier, resolvedDigest,
	), false
}

func generateBundleHash(ctx context.Context, bundlePath string) (string, error) {
	logger := logr.FromContextOrDiscard(ctx)
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

	artifactsWriter := artifacts.WriterFromContext(ctx)
	if artifactsWriter != nil {
		_, err := artifactsWriter.WriteFile("hashes.txt", &hashBuffer)
		if err != nil {
			return "", fmt.Errorf("could not write hash file to artifacts dir: %w", err)
		}
	}

	sum := fmt.Sprintf("%x", md5.Sum(hashBuffer.Bytes()))

	logger.V(log.DBG).Info("md5 sum", "md5sum", sum)

	return sum, nil
}

// Results will return the results of check execution.
func (c *CraneEngine) Results(ctx context.Context) certification.Results {
	return c.results
}

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func untar(ctx context.Context, dst string, r io.Reader) error {
	logger := logr.FromContextOrDiscard(ctx)
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
				logger.V(log.DBG).Info(fmt.Sprintf("Error creating link: %s. Ignoring.", header.Name))
				continue
			}
		}
	}
}

// writeCertImage takes imageRef and writes it to disk as JSON representing a pyxis.CertImage
// struct. The file is written at path certification.DefaultCertImageFilename.
//
//nolint:unparam // ctx is unused. Keep for future use.
func writeCertImage(ctx context.Context, imageRef image.ImageReference) error {
	logger := logr.FromContextOrDiscard(ctx)

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

	artifactWriter := artifacts.WriterFromContext(ctx)
	if artifactWriter != nil {
		fileName, err := artifactWriter.WriteFile(check.DefaultCertImageFilename, bytes.NewReader(certImageJSON))
		if err != nil {
			return fmt.Errorf("failed to save file to artifacts directory: %w", err)
		}

		logger.V(log.TRC).Info("image config written to disk", "filename", fileName)
	}

	return nil
}

func getBgName(srcrpm string) string {
	parts := strings.Split(srcrpm, "-")
	return strings.Join(parts[0:len(parts)-2], "-")
}

func writeRPMManifest(ctx context.Context, containerFSPath string) error {
	logger := logr.FromContextOrDiscard(ctx)
	pkgList, err := rpm.GetPackageList(ctx, containerFSPath)
	if err != nil {
		logger.Error(err, "could not get rpm list, continuing without it")
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
				logger.V(log.DBG).Info("string did not match the format required", "pgp", packageInfo.PGP)
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

	if artifactWriter := artifacts.WriterFromContext(ctx); artifactWriter != nil {
		fileName, err := artifactWriter.WriteFile(check.DefaultRPMManifestFilename, bytes.NewReader(rpmManifestJSON))
		if err != nil {
			return fmt.Errorf("failed to save file to artifacts directory: %w", err)
		}

		logger.V(log.TRC).Info("rpm manifest written to disk", "filename", fileName)
	}

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

// retryOnceAfter is a crane option that retries once after t duration.
func retryOnceAfter(t time.Duration) crane.Option {
	return func(o *crane.Options) {
		o.Remote = append(o.Remote, remote.WithRetryBackoff(remote.Backoff{
			Duration: t,
			Factor:   1.0,
			Jitter:   0.1,
			Steps:    2,
		}))
	}
}

// CheckEngine defines the functionality necessary to run all checks for a policy,
// and return the results of that check execution.
type CheckEngine interface {
	// ExecuteChecks should execute all checks in a policy and internally
	// store the results. Errors returned by ExecuteChecks should reflect
	// errors in pre-validation tasks, and not errors in individual check
	// execution itself.
	ExecuteChecks(context.Context) error
	// Results returns the outcome of executing all checks.
	Results(context.Context) certification.Results
}

func New(ctx context.Context,
	image string,
	checks []check.Check,
	kubeconfig []byte,
	dockerconfig string,
	isBundle,
	isScratch bool,
	insecure bool,
	platform string,
) (CheckEngine, error) {
	return &CraneEngine{
		Kubeconfig:   kubeconfig,
		DockerConfig: dockerconfig,
		Image:        image,
		Checks:       checks,
		IsBundle:     isBundle,
		IsScratch:    isScratch,
		Platform:     platform,
	}, nil
}

// OperatorCheckConfig contains configuration relevant to an individual check's execution.
type OperatorCheckConfig struct {
	ScorecardImage, ScorecardWaitTime, ScorecardNamespace, ScorecardServiceAccount string
	IndexImage, DockerConfig, Channel                                              string
	Kubeconfig                                                                     []byte
}

// InitializeOperatorChecks returns opeartor checks for policy p give cfg.
func InitializeOperatorChecks(ctx context.Context, p policy.Policy, cfg OperatorCheckConfig) ([]check.Check, error) {
	switch p {
	case policy.PolicyOperator:
		return []check.Check{
			operatorpol.NewScorecardBasicSpecCheck(operatorsdk.New(cfg.ScorecardImage, exec.Command), cfg.ScorecardNamespace, cfg.ScorecardServiceAccount, cfg.Kubeconfig, cfg.ScorecardWaitTime),
			operatorpol.NewScorecardOlmSuiteCheck(operatorsdk.New(cfg.ScorecardImage, exec.Command), cfg.ScorecardNamespace, cfg.ScorecardServiceAccount, cfg.Kubeconfig, cfg.ScorecardWaitTime),
			operatorpol.NewDeployableByOlmCheck(cfg.IndexImage, cfg.DockerConfig, cfg.Channel),
			operatorpol.NewValidateOperatorBundleCheck(),
			operatorpol.NewCertifiedImagesCheck(pyxis.NewPyxisClient(
				check.DefaultPyxisHost,
				"",
				"",
				&http.Client{Timeout: 60 * time.Second}),
			),
			operatorpol.NewSecurityContextConstraintsCheck(),
			&operatorpol.RelatedImagesCheck{},
		}, nil
	}

	return nil, fmt.Errorf("provided operator policy %s is unknown", p)
}

// ContainerCheckConfig contains configuration relevant to an individual check's execution.
type ContainerCheckConfig struct {
	DockerConfig, PyxisAPIToken, CertificationProjectID string
}

// InitializeContainerChecks returns the appropriate checks for policy p given cfg.
func InitializeContainerChecks(ctx context.Context, p policy.Policy, cfg ContainerCheckConfig) ([]check.Check, error) {
	switch p {
	case policy.PolicyContainer:
		return []check.Check{
			&containerpol.HasLicenseCheck{},
			containerpol.NewHasUniqueTagCheck(cfg.DockerConfig),
			&containerpol.MaxLayersCheck{},
			&containerpol.HasNoProhibitedPackagesCheck{},
			&containerpol.HasRequiredLabelsCheck{},
			&containerpol.RunAsNonRootCheck{},
			&containerpol.HasModifiedFilesCheck{},
			containerpol.NewBasedOnUbiCheck(pyxis.NewPyxisClient(
				check.DefaultPyxisHost,
				cfg.PyxisAPIToken,
				cfg.CertificationProjectID,
				&http.Client{Timeout: 60 * time.Second})),
		}, nil
	case policy.PolicyRoot:
		return []check.Check{
			&containerpol.HasLicenseCheck{},
			containerpol.NewHasUniqueTagCheck(cfg.DockerConfig),
			&containerpol.MaxLayersCheck{},
			&containerpol.HasNoProhibitedPackagesCheck{},
			&containerpol.HasRequiredLabelsCheck{},
			&containerpol.HasModifiedFilesCheck{},
			containerpol.NewBasedOnUbiCheck(pyxis.NewPyxisClient(
				check.DefaultPyxisHost,
				cfg.PyxisAPIToken,
				cfg.CertificationProjectID,
				&http.Client{Timeout: 60 * time.Second})),
		}, nil
	case policy.PolicyScratch:
		return []check.Check{
			&containerpol.HasLicenseCheck{},
			containerpol.NewHasUniqueTagCheck(cfg.DockerConfig),
			&containerpol.MaxLayersCheck{},
			&containerpol.HasRequiredLabelsCheck{},
			&containerpol.RunAsNonRootCheck{},
		}, nil
	}

	return nil, fmt.Errorf("provided container policy %s is unknown", p)
}

// makeCheckList returns a list of check names.
func makeCheckList(checks []check.Check) []string {
	checkNames := make([]string, len(checks))

	for i, check := range checks {
		checkNames[i] = check.Name()
	}

	return checkNames
}

// checkNamesFor produces a slice of names for checks in the requested policy.
func checkNamesFor(ctx context.Context, p policy.Policy) []string {
	var c []check.Check
	switch p {
	case policy.PolicyContainer, policy.PolicyRoot, policy.PolicyScratch:
		c, _ = InitializeContainerChecks(ctx, p, ContainerCheckConfig{})
	case policy.PolicyOperator:
		c, _ = InitializeOperatorChecks(ctx, p, OperatorCheckConfig{})
	default:
		return []string{}
	}

	return makeCheckList(c)
}

// OperatorPolicy returns the names of checks in the operator policy.
func OperatorPolicy(ctx context.Context) []string {
	return checkNamesFor(ctx, policy.PolicyOperator)
}

// ContainerPolicy returns the names of checks in the container policy.
func ContainerPolicy(ctx context.Context) []string {
	return checkNamesFor(ctx, policy.PolicyContainer)
}

// ScratchContainerPolicy returns the names of checks in the
// container policy with scratch exception.
func ScratchContainerPolicy(ctx context.Context) []string {
	return checkNamesFor(ctx, policy.PolicyScratch)
}

// RootExceptionContainerPolicy returns the names of checks in the
// container policy with root exception.
func RootExceptionContainerPolicy(ctx context.Context) []string {
	return checkNamesFor(ctx, policy.PolicyRoot)
}
