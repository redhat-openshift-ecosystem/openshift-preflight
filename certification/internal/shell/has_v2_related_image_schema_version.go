package shell

import (
	"encoding/json"
	"fmt"

	"github.com/operator-framework/api/pkg/manifests"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
)

// RelatedImagesAreSchemaVersion2Check is part of the Operator policy and implements
// the Check interface.
type RelatedImagesAreSchemaVersion2Check struct{}

// Validate checks the image manifest for each related image referenced in a
// ClusterServiceVersion and ensures that the schema version used is version 2.
func (p *RelatedImagesAreSchemaVersion2Check) Validate(image string) (bool, error) {
	imageToSchemaVersion, err := p.getDataToValidate(image)
	if err != nil {
		return false, fmt.Errorf("%w: %s", errors.ErrRunContainerFailed, err)
	}

	return p.validate(imageToSchemaVersion)
}

// getDataToValidate pulls a ClusterServiceVersion from the input operator bundle,
// checks the ClusterServiceVersion's related images declaration, and assembles a list of
// images and their image manifest schema version values.
func (p *RelatedImagesAreSchemaVersion2Check) getDataToValidate(image string) (map[string]int, error) {
	bundlePath, err := p.extractBundleFromImage(image)
	if err != nil {
		log.Error(err)
		return nil, fmt.Errorf("%w: %s", errors.ErrDeterminingRelatedImageSchemaVers, err)
	}
	csv, err := p.readBundle(bundlePath)
	if err != nil {
		log.Error(err)
		return nil, fmt.Errorf("%w: %s", errors.ErrDeterminingRelatedImageSchemaVers, err)
	}

	relatedImages, err := p.getRelatedImagesForCSV(csv)
	if err != nil {
		log.Error(err)
		return nil, fmt.Errorf("%w: %s", errors.ErrDeterminingRelatedImageSchemaVers, err)
	}

	imageSchemaVersionMap, err := p.inspectSchemaVersionForImage(relatedImages)
	if err != nil {
		log.Error(err)
		return nil, fmt.Errorf("%w: %s", errors.ErrDeterminingRelatedImageSchemaVers, err)
	}

	return imageSchemaVersionMap, nil
}

// extractBundleFromImage will extract the bundle contents from the specified image, and return the
// path to the bundle on the filesystem. This is done by creating the container, copying the contents
// out of the container, and then cleaning up the container.
func (p *RelatedImagesAreSchemaVersion2Check) extractBundleFromImage(image string) (string, error) {
	localBundlePath := "extracted-bundle"

	// create the container. Bundle containers are non-runnable, so we need to
	// add a dummy entrypoint so that we can create the container.
	createReport, err := podmanEngine.Create(image, &cli.PodmanCreateOptions{Entrypoint: "false"})
	if err != nil {
		log.Error("stdout: ", createReport.Stdout)
		log.Error("stderr: ", createReport.Stderr)
		return "", err
	}

	defer func() {
		deleteReport, err := podmanEngine.Remove(createReport.ContainerID)
		if err != nil {
			log.Warn("a non-fatal error occurred while trying to remove container: ", err)
			log.Warn("stdout: ", deleteReport.Stdout)
			log.Warn("stderr: ", deleteReport.Stderr)
		}
	}()

	copyReport, err := podmanEngine.CopyFrom(createReport.ContainerID, "/", localBundlePath)
	if err != nil {
		log.Error("stdout: ", copyReport.Stdout)
		log.Error("stderr: ", copyReport.Stderr)
		return "", err
	}

	return localBundlePath, nil
}

// readBundle will accept the manifests directory where a bundle is expected to live,
// and walks the directory to find all bundle assets.
func (p *RelatedImagesAreSchemaVersion2Check) readBundle(manifestsDir string) (*operatorsv1alpha1.ClusterServiceVersion, error) {
	bundle, err := manifests.GetBundleFromDir(manifestsDir)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrDeterminingRelatedImageSchemaVers, err)
	}

	return bundle.CSV, nil
}

// getRelatedImagesForCSV will return a slice of strings containing the images found in the relatedImages field of
// the input ClusterServiceVersion.
func (p *RelatedImagesAreSchemaVersion2Check) getRelatedImagesForCSV(csv *operatorsv1alpha1.ClusterServiceVersion) ([]string, error) {
	relatedImages := csv.Spec.RelatedImages

	// no related images == nothing to check
	if len(relatedImages) == 0 {
		return nil, nil
	}

	imageStrings := make([]string, len(relatedImages))
	for i, relatedImage := range relatedImages {
		imageStrings[i] = relatedImage.Image
	}

	return imageStrings, nil
}

// inspectSchemaVersionForImages will inspect each input image using skopeo and returns a map of each image
// with its corresponding schemaVersion. It is expeted that len(images) > 0.
func (p *RelatedImagesAreSchemaVersion2Check) inspectSchemaVersionForImage(images []string) (map[string]int, error) {
	inspectOpts := cli.SkopeoInspectOptions{Raw: true}

	imageToSchemaVersion := map[string]int{}
	for _, image := range images {
		log.Debug("inspecting image ", image)
		report, err := skopeoEngine.InspectImage(image, inspectOpts)
		if err != nil {
			log.Error("encountered an error while inspecting")
			log.Error("stdout: ", report.Stdout)
			log.Error("stderr: ", report.Stderr)
			return nil, err
		}

		schemaVersion, err := p.getSchemaVersionFromRawManifest(report.Blob)
		if err != nil {
			return nil, err
		}

		imageToSchemaVersion[image] = schemaVersion
	}

	return imageToSchemaVersion, nil
}

// getSchemaVersionFromRawManifest accepts the raw manifest blob, asserts that it is the expected format,
// and returns the value of the top-level schemaVersion key.
//
// TODO: If a struct representation of the raw manifest can be found, this should be used instead. We only
// unmarshal to a map[string]interface here because schemaVersion is a top-level key, implying that we
// only need a single assertion to reach our desired value.
func (p *RelatedImagesAreSchemaVersion2Check) getSchemaVersionFromRawManifest(manifest []byte) (int, error) {
	var rawManifest map[string]interface{}
	if err := json.Unmarshal(manifest, &rawManifest); err != nil {
		log.Error("unable to unmarshal inspected manifest")
		return 0, err
	}

	schemaVersionIface, found := rawManifest["schemaVersion"]
	if !found {
		return 0, fmt.Errorf("rawManifest is an unexpected format")
	}

	// when unmarshaled by json, number will be a float64
	schemaVersionf64, ok := schemaVersionIface.(float64)
	if !ok {
		return 0, fmt.Errorf("schemaVersion value is an unexpected type")
	}

	// We don't expect the schemaVersion value to be large, so converting to
	// an int should be okay.
	schemaVersion := int(schemaVersionf64)

	return schemaVersion, nil
}

// validate checks each image's schema value and returns true if they're all version 2.
func (p *RelatedImagesAreSchemaVersion2Check) validate(imageToSchemaVersionMap map[string]int) (bool, error) {

	// if the map is empty or nil, then we auto pass this check.
	// note that internally, len(nil) == 0 according to staticcheck
	if len(imageToSchemaVersionMap) == 0 {
		return true, nil
	}

	v2ImageManifests := []string{}
	notv2ImageManifests := []string{}

	for image, schemaVersion := range imageToSchemaVersionMap {
		if schemaVersion == 2 {
			v2ImageManifests = append(v2ImageManifests, image)
		} else {
			notv2ImageManifests = append(notv2ImageManifests, image)
		}
	}

	// if we determine that all images are using schemaVersion 2, then we pass the check
	if len(v2ImageManifests) == len(imageToSchemaVersionMap) {
		return true, nil
	}

	log.Info("the following related images found in CSV were not image manifests with schema version 2", notv2ImageManifests)
	return false, nil
}

func (p *RelatedImagesAreSchemaVersion2Check) Name() string {
	return "RelatedImagesAreSchemaVersion2"
}

func (p *RelatedImagesAreSchemaVersion2Check) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "An operator bundle's relatedImages must be accessible at image manifest schema version 2",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *RelatedImagesAreSchemaVersion2Check) Help() certification.HelpText {
	return certification.HelpText{
		// TODO
		Message: "Check RelatedImagesAreSchemaVersion2 has encountered an error. Please review the preflight.log for more information.",
		Suggestion: "Ensure that the related images listed in your ClusterServiceVersion contain images respecting schema version 2 " +
			"as defined here: https://docs.docker.com/registry/spec/manifest-v2-2/",
	}
}
