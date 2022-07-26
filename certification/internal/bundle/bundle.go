package bundle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/blang/semver"
	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/operatorsdk"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

const ocpVerV1beta1Unsupported = "4.9"

// versionsKey is the OpenShift versions in annotations.yaml that lists the versions allowed for an operator
const versionsKey = "com.redhat.openshift.versions"

type operatorSdk interface {
	BundleValidate(context.Context, string, operatorsdk.OperatorSdkBundleValidateOptions) (*operatorsdk.OperatorSdkBundleValidateReport, error)
}

func Validate(ctx context.Context, operatorSdk operatorSdk, imagePath string) (*operatorsdk.OperatorSdkBundleValidateReport, error) {
	selector := []string{"community", "operatorhub"}
	opts := operatorsdk.OperatorSdkBundleValidateOptions{
		Selector:        selector,
		Verbose:         true,
		ContainerEngine: "none",
		OutputFormat:    "json-alpha1",
	}

	log.Trace("reading annotations file from the bundle")
	log.Debug("image extraction directory is ", imagePath)
	// retrieve the operator metadata from bundle image
	annotationsFileName := filepath.Join(imagePath, "metadata", "annotations.yaml")
	annotationsFile, err := os.Open(annotationsFileName)
	if err != nil {
		return nil, fmt.Errorf("could not open annotations.yaml: %v", err)
	}
	annotations, err := GetAnnotations(ctx, annotationsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to get annotations.yaml from the bundle: %v", err)
	}

	if versions, ok := annotations[versionsKey]; ok {
		// Check that the label range contains >= 4.9
		if isTarget49OrGreater(versions) {
			log.Debug("OpenShift 4.9 detected in annotations. Running with additional checks enabled.")
			opts.OptionalValues = make(map[string]string)
			opts.OptionalValues["k8s-version"] = "1.22"
		}
	}

	return operatorSdk.BundleValidate(ctx, imagePath, opts)
}

func isTarget49OrGreater(ocpLabelIndex string) bool {
	semVerOCPV1beta1Unsupported, _ := semver.ParseTolerant(ocpVerV1beta1Unsupported)
	// the OCP range informed cannot allow carry on to OCP 4.9+
	beginsEqual := strings.HasPrefix(ocpLabelIndex, "=")
	// It means that the OCP label is =OCP version
	if beginsEqual {
		version := cleanStringToGetTheVersionToParse(strings.Split(ocpLabelIndex, "=")[1])
		verParsed, err := semver.ParseTolerant(version)
		if err != nil {
			log.Errorf("unable to parse the value (%s) on (%s)", version, ocpLabelIndex)
			return false
		}

		if verParsed.GE(semVerOCPV1beta1Unsupported) {
			return true
		}
		return false
	}
	indexRange := cleanStringToGetTheVersionToParse(ocpLabelIndex)
	if len(indexRange) > 1 {
		// Bare version
		if !strings.Contains(indexRange, "-") {
			verParsed, err := semver.ParseTolerant(indexRange)
			if err != nil {
				log.Error("unable to parse the version")
				return false
			}
			if verParsed.GE(semVerOCPV1beta1Unsupported) {
				return true
			}
		}

		versions := strings.Split(indexRange, "-")
		version := versions[0]
		if len(versions) > 1 {
			version = versions[1]
			verParsed, err := semver.ParseTolerant(version)
			if err != nil {
				log.Error("unable to parse the version")
				return false
			}

			if verParsed.GE(semVerOCPV1beta1Unsupported) {
				return true
			}
			return false
		}

		verParsed, err := semver.ParseTolerant(version)
		if err != nil {
			log.Error("unable to parse the version")
			return false
		}

		if semVerOCPV1beta1Unsupported.GE(verParsed) {
			return true
		}
	}
	return false
}

// cleanStringToGetTheVersionToParse will remove the expected characters for
// we are able to parse the version informed.
func cleanStringToGetTheVersionToParse(value string) string {
	doubleQuote := "\""
	singleQuote := "'"
	value = strings.ReplaceAll(value, singleQuote, "")
	value = strings.ReplaceAll(value, doubleQuote, "")
	value = strings.ReplaceAll(value, "v", "")
	return value
}

// GetAnnotations accepts a context, and an io.Reader that is expected to provide
// the annotations.yaml, and parses the annotations from there
func GetAnnotations(ctx context.Context, r io.Reader) (map[string]string, error) {
	fileContents, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("fail to read metadata/annotation.yaml file in bundle: %v", err)
	}

	annotations, err := ExtractAnnotationsBytes(ctx, fileContents)
	if err != nil {
		return nil, fmt.Errorf("metadata/annotations.yaml found but is malformed: %v", err)
	}

	return annotations, nil
}

// extractAnnotationsBytes reads the annotation data read from a file and returns the expected format for that yaml
// represented as a map[string]string.
func ExtractAnnotationsBytes(ctx context.Context, annotationBytes []byte) (map[string]string, error) {
	type metadata struct {
		Annotations map[string]string
	}

	if len(annotationBytes) == 0 {
		return nil, errors.New("the annotations file was empty")
	}

	var bundleMeta metadata
	if err := yaml.Unmarshal(annotationBytes, &bundleMeta); err != nil {
		return nil, fmt.Errorf("metadata/annotations.yaml found but is malformed: %v", err)
	}

	return bundleMeta.Annotations, nil
}

func GetCsvFilePathFromBundle(imageDir string) (string, error) {
	log.Trace("reading clusterserviceversion file from the bundle")
	log.Debug("image directory is ", imageDir)
	matches, err := filepath.Glob(filepath.Join(imageDir, "manifests", "*.clusterserviceversion.yaml"))
	if err != nil {
		return "", fmt.Errorf("glob pattern is malformed: %v", err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("unable to find clusterserviceversion file in the bundle image: %v", os.ErrNotExist)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("more than one CSV file detected in bundle")
	}
	log.Debugf("The path to csv file is %s", matches[0])
	return matches[0], nil
}

func csvFromReader(ctx context.Context, csvReader io.Reader) (*operatorv1alpha1.ClusterServiceVersion, error) {
	var csv operatorv1alpha1.ClusterServiceVersion
	bts, err := io.ReadAll(csvReader)
	if err != nil {
		return nil, fmt.Errorf("could not get CSV from reader: %v", err)
	}
	err = yaml.Unmarshal(bts, &csv)
	if err != nil {
		return nil, fmt.Errorf("malformed CSV detected: %v", err)
	}

	return &csv, nil
}

func GetSupportedInstallModes(ctx context.Context, csvReader io.Reader) (map[string]bool, error) {
	csv, err := csvFromReader(ctx, csvReader)
	if err != nil {
		return nil, err
	}

	installedModes := make(map[string]bool, len(csv.Spec.InstallModes))
	for _, v := range csv.Spec.InstallModes {
		if v.Supported {
			installedModes[string(v.Type)] = true
		}
	}
	return installedModes, nil
}
