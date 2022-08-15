package bundle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/blang/semver"
	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/operatorsdk"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

// versionsKey is the OpenShift versions in annotations.yaml that lists the versions allowed for an operator
const versionsKey = "com.redhat.openshift.versions"

// This table signifies what the NEXT release of OpenShift will
// deprecate, not what it matches up to.
var ocpToKubeVersion = map[string]string{
	"4.9":  "1.22",
	"4.10": "1.23",
	"4.11": "1.24",
	"4.12": "1.25",
	"4.13": "1.26",
}

const latestReleasedVersion = "4.11"

type operatorSdk interface {
	BundleValidate(context.Context, string, operatorsdk.OperatorSdkBundleValidateOptions) (*operatorsdk.OperatorSdkBundleValidateReport, error)
}

func Validate(ctx context.Context, operatorSdk operatorSdk, imagePath string) (*operatorsdk.OperatorSdkBundleValidateReport, error) {
	selector := []string{"community", "operatorhub", "alpha-deprecated-apis"}
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
		targetVersion, err := targetVersion(versions)
		if err != nil {
			// Could not parse the version, which probably means the annotation is invalid
			return nil, fmt.Errorf("%v", err)
		}
		if k8sVer, ok := ocpToKubeVersion[targetVersion]; ok {
			log.Debugf("OpenShift %s detected in annotations. Running with additional checks enabled.", targetVersion)
			opts.OptionalValues = make(map[string]string)
			opts.OptionalValues["k8s-version"] = k8sVer
		}
	}

	return operatorSdk.BundleValidate(ctx, imagePath, opts)
}

func targetVersion(ocpLabelIndex string) (string, error) {
	beginsEqual := strings.HasPrefix(ocpLabelIndex, "=")
	// It means that the OCP label is =OCP version
	if beginsEqual {
		version := cleanStringToGetTheVersionToParse(strings.Split(ocpLabelIndex, "=")[1])
		verParsed, err := semver.ParseTolerant(version)
		if err != nil {
			return "", fmt.Errorf("unable to parse the value (%s) on (%s): %v", version, ocpLabelIndex, err)
		}

		return fmt.Sprintf("%d.%d", verParsed.Major, verParsed.Minor), nil
	}

	indexRange := cleanStringToGetTheVersionToParse(ocpLabelIndex)
	if len(indexRange) > 1 {
		// Bare version, so send back latest released
		if !strings.Contains(indexRange, "-") {
			verParsed, err := semver.ParseTolerant(indexRange)
			if err != nil {
				// The passed version is not valid. We don't care what it is,
				// just that it's valid.
				return "", fmt.Errorf("unable to parse the version: %v", err)
			}

			// If the specified version is greater than latestReleased, we will accept that
			latestReleasedParsed, _ := semver.ParseTolerant(latestReleasedVersion)
			if verParsed.GT(latestReleasedParsed) {
				return fmt.Sprintf("%d.%d", verParsed.Major, verParsed.Minor), nil
			}

			return latestReleasedVersion, nil
		}

		versions := strings.Split(indexRange, "-")
		// This is a normal range of 1.0-2.0
		if len(versions) > 1 && versions[1] != "" {
			version := versions[1]
			verParsed, err := semver.ParseTolerant(version)
			if err != nil {
				return "", fmt.Errorf("unable to parse the version: %v", err)
			}
			return fmt.Sprintf("%d.%d", verParsed.Major, verParsed.Minor), nil
		}

		// This is an open-ended range: v1-. This is not valid.
		// So, we just fall through to the default return.
		return "", fmt.Errorf("unable to parse the version: malformed range: %s", indexRange)
	}
	return "", fmt.Errorf("unable to parse the version: unknown error")
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

// GetSecurityContextConstraints returns an string array of SCC resource names requested by the operator as specified
// in the csv
func GetSecurityContextConstraints(ctx context.Context, csvReader io.Reader) ([]string, error) {
	var csv operatorv1alpha1.ClusterServiceVersion
	bts, err := io.ReadAll(csvReader)
	if err != nil {
		return nil, fmt.Errorf("could not get CSV from reader: %v", err)
	}
	err = yaml.Unmarshal(bts, &csv)
	if err != nil {
		return nil, fmt.Errorf("malformed CSV detected: %v", err)
	}
	for _, cp := range csv.Spec.InstallStrategy.StrategySpec.ClusterPermissions {
		for _, rule := range cp.Rules {
			if hasSCCApiGroup(rule) && hasSCCResource(rule) {
				return rule.ResourceNames, nil
			}
		}
	}
	return nil, nil
}

// hasSCCApiGroup returns a bool indicating if security.openshift.io is in the list of apigroups referenced in a policy
// rule
func hasSCCApiGroup(rule rbacv1.PolicyRule) bool {
	for _, apiGroup := range rule.APIGroups {
		if apiGroup == "security.openshift.io" {
			return true
		}
	}
	return false
}

// hasSCCResource returns a bool indicating if any securitycontextconstraints resources are referenced in a policy rule
func hasSCCResource(rule rbacv1.PolicyRule) bool {
	for _, resource := range rule.Resources {
		if resource == "securitycontextconstraints" {
			return true
		}
	}
	return false
}
