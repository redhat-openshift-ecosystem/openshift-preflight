package bundle

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"

	"github.com/blang/semver"
	"github.com/go-logr/logr"
	"github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/validation"

	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/yaml"
)

// This table signifies what the NEXT release of OpenShift will
// deprecate, not what it matches up to.
var ocpToKubeVersion = map[string]string{
	"4.9":  "1.22",
	"4.10": "1.23",
	"4.11": "1.24",
	"4.12": "1.25",
	"4.13": "1.26",
	"4.14": "1.27",
	"4.15": "1.28",
	"4.16": "1.29",
	"4.17": "1.30",
	"4.18": "1.31",
	"4.19": "1.32",
	"4.20": "1.33",
	"4.21": "1.34",
	"4.22": "1.35",
}

const latestReleasedVersion = "4.21"

var BundleFiles = []string{
	"/manifests/*",
	"/metadata/annotations.yaml",
}

func Validate(ctx context.Context, imagePath string) (*Report, error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(log.TRC).Info("reading annotations file from the bundle")
	logger.V(log.DBG).Info("image extraction directory", "directory", imagePath)

	bundle, err := manifests.GetBundleFromDir(imagePath)
	if err != nil {
		return nil, fmt.Errorf("could not load bundle from path: %s: %v", imagePath, err)
	}
	validators := validation.DefaultBundleValidators.WithValidators(
		validation.AlphaDeprecatedAPIsValidator,
		validation.OperatorHubV2Validator,
		validation.StandardCapabilitiesValidator,
		validation.StandardCategoriesValidator,
	)

	objs := bundle.ObjectsToValidate()

	// retrieve the operator metadata from bundle image
	annotationsFileName := filepath.Join(imagePath, "metadata", "annotations.yaml")
	annotationsFile, err := os.Open(annotationsFileName)
	if err != nil {
		return nil, fmt.Errorf("could not open annotations.yaml: %v", err)
	}
	annotations, err := LoadAnnotations(ctx, annotationsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to get annotations.yaml from the bundle: %v", err)
	}

	optionalValues := make(map[string]string)
	if annotations.OpenshiftVersions != "" {
		// Check that the label range contains >= 4.9
		targetVersion, err := targetVersion(annotations.OpenshiftVersions)
		if err != nil {
			// Could not parse the version, which probably means the annotation is invalid
			return nil, fmt.Errorf("%v", err)
		}
		if k8sVer, found := ocpToKubeVersion[targetVersion]; found {
			logger.V(log.DBG).Info("running with additional checks enabled because of the OpenShift version detected", "version", targetVersion)
			optionalValues = make(map[string]string)
			optionalValues["k8s-version"] = k8sVer
		}
	}
	objs = append(objs, optionalValues)

	results := validators.Validate(objs...)
	passed := true
	for _, v := range results {
		if v.HasError() {
			passed = false
			break
		}
	}

	return &Report{Results: results, Passed: passed}, nil
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

// LoadAnnotations reads an operator bundle's annotations.yaml from r.
func LoadAnnotations(ctx context.Context, r io.Reader) (*Annotations, error) {
	annFile, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("fail to read metadata/annotation.yaml file in bundle: %v", err)
	}

	if len(annFile) == 0 {
		return nil, fmt.Errorf("annotations file was empty")
	}

	var annotationsFile AnnotationsFile
	if err := yaml.Unmarshal(annFile, &annotationsFile); err != nil {
		return nil, fmt.Errorf("unable to load the annotations file: %v", err)
	}

	return &annotationsFile.Annotations, nil
}

// GetSecurityContextConstraints returns an string array of SCC resource names requested by the operator as specified
// in the csv
func GetSecurityContextConstraints(ctx context.Context, bundlePath string) ([]string, error) {
	bundle, err := manifests.GetBundleFromDir(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("could not get bundle from dir: %s: %v", bundlePath, err)
	}
	for _, cp := range bundle.CSV.Spec.InstallStrategy.StrategySpec.ClusterPermissions {
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
