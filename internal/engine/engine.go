package engine

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/go-logr/logr"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/openshift"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/operatorsdk"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/option"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy"
	containerpol "github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy/container"
	operatorpol "github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy/operator"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/pyxis"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/rpm"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/runtime"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/cache"
)

// New creates a new CraneEngine from the passed params
func New(ctx context.Context,
	checks []check.Check,
	kubeconfig []byte,
	cfg runtime.Config,
) (craneEngine, error) {
	return craneEngine{
		kubeconfig:         kubeconfig,
		dockerConfig:       cfg.DockerConfig,
		image:              cfg.Image,
		checks:             checks,
		isBundle:           cfg.Bundle,
		isScratch:          cfg.Scratch,
		platform:           cfg.Platform,
		insecure:           cfg.Insecure,
		manifestListDigest: cfg.ManifestListDigest,
	}, nil
}

// CraneEngine implements a certification.CheckEngine, and leverage crane to interact with
// the container registry and target image.
type craneEngine struct {
	// Kubeconfig is a byte slice containing a valid Kubeconfig to be used by checks.
	kubeconfig []byte
	// DockerConfig is the credential required to pull the image.
	dockerConfig string
	// Image is what is being tested, and should contain the
	// fully addressable path (including registry, namespaces, etc)
	// to the image
	image string
	// Checks is an array of all checks to be executed against
	// the image provided.
	checks []check.Check
	// Platform is the container platform to use. E.g. amd64.
	platform string

	// IsBundle is an indicator that the asset is a bundle.
	isBundle bool

	// IsScratch is an indicator that the asset is a scratch image
	isScratch bool

	// Insecure controls whether to allow an insecure connection to
	// the registry crane connects with.
	insecure bool

	// ManifestListDigest is the sha256 digest for the manifest list
	manifestListDigest string

	imageRef image.ImageReference
	results  certification.Results
}

func (c *craneEngine) CranePlatform() string {
	return c.platform
}

func (c *craneEngine) CraneDockerConfig() string {
	return c.dockerConfig
}

func (c *craneEngine) CraneInsecure() bool {
	return c.insecure
}

var _ option.CraneConfig = &craneEngine{}

func (c *craneEngine) ExecuteChecks(ctx context.Context) error {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("target image", "image", c.image)

	// pull the image and save to fs
	logger.V(log.DBG).Info("pulling image from target registry")
	options := option.GenerateCraneOptions(ctx, c)
	img, err := crane.Pull(c.image, options...)
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

	requiredFilePatternsCount := 0
	for _, check := range c.checks {
		requiredFilePatternsCount += len(check.RequiredFilePatterns())
	}

	requiredFilePatterns := make([]string, 0, requiredFilePatternsCount)
	for _, check := range c.checks {
		requiredFilePatterns = append(requiredFilePatterns, check.RequiredFilePatterns()...)
	}
	for i, pattern := range requiredFilePatterns {
		requiredFilePatterns[i] = strings.TrimLeft(pattern, "/")
	}

	slices.Sort(requiredFilePatterns)
	requiredFilePatterns = slices.Compact(requiredFilePatterns)

	if err := untar(ctx, containerFSPath, img, requiredFilePatterns); err != nil {
		return err
	}

	reference, err := name.ParseReference(c.image)
	if err != nil {
		return fmt.Errorf("image uri could not be parsed: %v", err)
	}

	// store the image internals in the engine image reference to pass to validations.
	c.imageRef = image.ImageReference{
		ImageURI:           c.image,
		ImageFSPath:        containerFSPath,
		ImageInfo:          img,
		ImageRegistry:      reference.Context().RegistryStr(),
		ImageRepository:    reference.Context().RepositoryStr(),
		ImageTagOrSha:      reference.Identifier(),
		ManifestListDigest: c.manifestListDigest,
	}

	if err := writeCertImage(ctx, c.imageRef); err != nil {
		return fmt.Errorf("could not write cert image: %v", err)
	}

	if !c.isScratch {
		if err := writeRPMManifest(ctx, containerFSPath); err != nil {
			return fmt.Errorf("could not write rpm manifest: %v", err)
		}
	}

	if c.isBundle {
		// Record test cluster version
		version, err := openshift.GetOpenshiftClusterVersion(ctx, c.kubeconfig)
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
	for _, executedCheck := range c.checks {
		logger := logger.WithValues("check", executedCheck.Name())
		ctx := logr.NewContext(ctx, logger)
		c.results.TestedImage = c.image

		logger.V(log.DBG).Info("running check")
		if executedCheck.Metadata().Level == check.LevelOptional || executedCheck.Metadata().Level == check.LevelWarn {
			logger.Info(fmt.Sprintf("Check %s is not currently being enforced.", executedCheck.Name()))
		}

		// run the validation
		checkStartTime := time.Now()
		checkPassed, err := executedCheck.Validate(ctx, c.imageRef)
		checkElapsedTime := time.Since(checkStartTime)

		if err != nil {
			logger.WithValues("result", "ERROR", "err", err.Error()).Info("check completed")
			result := certification.Result{Check: executedCheck, ElapsedTime: checkElapsedTime}
			c.results.Errors = appendUnlessOptional(c.results.Errors, *result.WithError(err))
			continue
		}

		if !checkPassed {
			// if a test doesn't pass but is of level warn include it in warning results, instead of failed results
			if executedCheck.Metadata().Level == check.LevelWarn {
				logger.WithValues("result", "WARNING").Info("check completed")
				c.results.Warned = appendUnlessOptional(c.results.Warned, certification.Result{Check: executedCheck, ElapsedTime: checkElapsedTime})
				continue
			}
			logger.WithValues("result", "FAILED").Info("check completed")
			c.results.Failed = appendUnlessOptional(c.results.Failed, certification.Result{Check: executedCheck, ElapsedTime: checkElapsedTime})
			continue
		}

		logger.WithValues("result", "PASSED").Info("check completed")
		c.results.Passed = appendUnlessOptional(c.results.Passed, certification.Result{Check: executedCheck, ElapsedTime: checkElapsedTime})
	}

	if len(c.results.Errors) > 0 || len(c.results.Failed) > 0 {
		c.results.PassedOverall = false
	} else {
		c.results.PassedOverall = true
	}

	if c.isBundle { // for operators:
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
		`This image's tag %s will be paired with digest %s `+
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

	keys := slices.Collect(maps.Keys(files))
	slices.Sort(keys)

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
func (c *craneEngine) Results(ctx context.Context) certification.Results {
	return c.results
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

		written, err := func() (int64, error) {
			uncompressed, err := layer.Uncompressed()
			if err != nil {
				return 0, fmt.Errorf("could not get uncompressed layer: %w", err)
			}
			defer uncompressed.Close()

			written, err := io.Copy(io.Discard, uncompressed)
			if err != nil {
				return written, fmt.Errorf("could not copy from layer: %w", err)
			}

			return written, nil
		}()
		if err != nil {
			return err
		}

		pyxisLayer := pyxis.Layer{
			LayerID: diffid.String(),
			Size:    written,
		}
		layerSizes = append(layerSizes, pyxisLayer)
	}

	manifestLayers := make([]string, 0, len(manifest.Layers))

	// CertImage expects the layers to be stored in the order from base to top.
	// Index 0 is the base layer, and the last index is the top layer.
	for _, layer := range slices.Backward(manifest.Layers) {
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
		PushDate:           addedDate,
		Registry:           imageRef.ImageRegistry,
		Repository:         imageRef.ImageRepository,
		Tags:               tags,
		ManifestListDigest: imageRef.ManifestListDigest,
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
	rpmSuffixRegexp, err := regexp.Compile("(-[0-9].*)")
	if err != nil {
		return fmt.Errorf("error while compiling regexp: %w", err)
	}
	pgpKeyIdRegexp, err := regexp.Compile(".*, Key ID (.*)")
	if err != nil {
		return fmt.Errorf("error while compiling regexp: %w", err)
	}
	for _, packageInfo := range pkgList {
		var bgName, endChop, srpmNevra, pgpKeyID string

		// accounting for the fact that not all packages have a source rpm
		if len(packageInfo.SourceRpm) > 0 {
			bgName = getBgName(packageInfo.SourceRpm)
			endChop = strings.TrimPrefix(strings.TrimSuffix(rpmSuffixRegexp.FindString(packageInfo.SourceRpm), ".rpm"), "-")

			srpmNevra = fmt.Sprintf("%s-%d:%s", bgName, packageInfo.Epoch, endChop)
		}

		if len(packageInfo.PGP) > 0 {
			matches := pgpKeyIdRegexp.FindStringSubmatch(packageInfo.PGP)
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

// OperatorCheckConfig contains configuration relevant to an individual check's execution.
type OperatorCheckConfig struct {
	ScorecardImage, ScorecardWaitTime, ScorecardNamespace, ScorecardServiceAccount string
	IndexImage, DockerConfig, Channel                                              string
	Kubeconfig                                                                     []byte
	CSVTimeout                                                                     time.Duration
	SubscriptionTimeout                                                            time.Duration
}

// InitializeOperatorChecks returns opeartor checks for policy p give cfg.
func InitializeOperatorChecks(ctx context.Context, p policy.Policy, cfg OperatorCheckConfig) ([]check.Check, error) {
	switch p {
	case policy.PolicyOperator:
		return []check.Check{
			operatorpol.NewScorecardBasicSpecCheck(operatorsdk.New(cfg.ScorecardImage, exec.Command), cfg.ScorecardNamespace, cfg.ScorecardServiceAccount, cfg.Kubeconfig, cfg.ScorecardWaitTime),
			operatorpol.NewScorecardOlmSuiteCheck(operatorsdk.New(cfg.ScorecardImage, exec.Command), cfg.ScorecardNamespace, cfg.ScorecardServiceAccount, cfg.Kubeconfig, cfg.ScorecardWaitTime),
			operatorpol.NewDeployableByOlmCheck(cfg.IndexImage, cfg.DockerConfig, cfg.Channel, operatorpol.WithCSVTimeout(cfg.CSVTimeout), operatorpol.WithSubscriptionTimeout(cfg.SubscriptionTimeout)),
			operatorpol.NewValidateOperatorBundleCheck(),
			operatorpol.NewCertifiedImagesCheck(pyxis.NewPyxisClient(
				check.DefaultPyxisHost,
				"",
				"",
				&http.Client{Timeout: 60 * time.Second}),
			),
			operatorpol.NewSecurityContextConstraintsCheck(),
			&operatorpol.RelatedImagesCheck{},
			operatorpol.FollowsRestrictedNetworkEnablementGuidelines{},
			operatorpol.RequiredAnnotations{},
		}, nil
	}

	return nil, fmt.Errorf("provided operator policy %s is unknown", p)
}

// ContainerCheckConfig contains configuration relevant to an individual check's execution.
type ContainerCheckConfig struct {
	DockerConfig, PyxisAPIToken, CertificationProjectID, PyxisHost string
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
			&containerpol.HasNoProhibitedLabelsCheck{},
			&containerpol.RunAsNonRootCheck{},
			&containerpol.HasModifiedFilesCheck{},
			containerpol.NewBasedOnUbiCheck(pyxis.NewPyxisClient(
				cfg.PyxisHost,
				cfg.PyxisAPIToken,
				cfg.CertificationProjectID,
				&http.Client{Timeout: 60 * time.Second})),
			&containerpol.HasProhibitedContainerName{},
		}, nil
	case policy.PolicyRoot:
		return []check.Check{
			&containerpol.HasLicenseCheck{},
			containerpol.NewHasUniqueTagCheck(cfg.DockerConfig),
			&containerpol.MaxLayersCheck{},
			&containerpol.HasNoProhibitedPackagesCheck{},
			&containerpol.HasRequiredLabelsCheck{},
			&containerpol.HasNoProhibitedLabelsCheck{},
			&containerpol.HasModifiedFilesCheck{},
			containerpol.NewBasedOnUbiCheck(pyxis.NewPyxisClient(
				cfg.PyxisHost,
				cfg.PyxisAPIToken,
				cfg.CertificationProjectID,
				&http.Client{Timeout: 60 * time.Second})),
			&containerpol.HasProhibitedContainerName{},
		}, nil
	case policy.PolicyScratchNonRoot:
		return []check.Check{
			&containerpol.HasLicenseCheck{},
			containerpol.NewHasUniqueTagCheck(cfg.DockerConfig),
			&containerpol.MaxLayersCheck{},
			&containerpol.HasRequiredLabelsCheck{},
			&containerpol.HasNoProhibitedLabelsCheck{},
			&containerpol.RunAsNonRootCheck{},
			&containerpol.HasProhibitedContainerName{},
		}, nil
	case policy.PolicyScratchRoot:
		return []check.Check{
			&containerpol.HasLicenseCheck{},
			containerpol.NewHasUniqueTagCheck(cfg.DockerConfig),
			&containerpol.MaxLayersCheck{},
			&containerpol.HasRequiredLabelsCheck{},
			&containerpol.HasNoProhibitedLabelsCheck{},
			&containerpol.HasProhibitedContainerName{},
		}, nil
	case policy.PolicyKonflux:
		return []check.Check{
			&containerpol.HasLicenseCheck{},
			containerpol.NewHasUniqueTagCheck(cfg.DockerConfig),
			&containerpol.MaxLayersCheck{},
			&containerpol.HasNoProhibitedPackagesCheck{},
			&containerpol.HasRequiredLabelsCheck{},
			&containerpol.RunAsNonRootCheck{},
			&containerpol.HasModifiedFilesCheck{},
			containerpol.NewBasedOnUbiCheck(pyxis.NewPyxisClient(
				cfg.PyxisHost,
				cfg.PyxisAPIToken,
				cfg.CertificationProjectID,
				&http.Client{Timeout: 60 * time.Second})),
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
	case policy.PolicyContainer, policy.PolicyRoot, policy.PolicyScratchNonRoot, policy.PolicyScratchRoot, policy.PolicyKonflux:
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

// ScratchNonRootContainerPolicy returns the names of checks in the
// container policy with scratch exception.
func ScratchNonRootContainerPolicy(ctx context.Context) []string {
	return checkNamesFor(ctx, policy.PolicyScratchNonRoot)
}

// ScratchRootContainerPolicy returns the names of checks in the
// container policy with scratch and root exception.
func ScratchRootContainerPolicy(ctx context.Context) []string {
	return checkNamesFor(ctx, policy.PolicyScratchRoot)
}

// RootExceptionContainerPolicy returns the names of checks in the
// container policy with root exception.
func RootExceptionContainerPolicy(ctx context.Context) []string {
	return checkNamesFor(ctx, policy.PolicyRoot)
}

// KonfluxContainerPolicy returns the names of checks to be used in
// a konflux pipeline
func KonfluxContainerPolicy(ctx context.Context) []string {
	return checkNamesFor(ctx, policy.PolicyKonflux)
}
