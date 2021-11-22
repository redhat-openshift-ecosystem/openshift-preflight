package operator

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func getAnnotationsFromBundle(mountedDir string) (map[string]string, error) {
	log.Trace("reading annotations file from the bundle")
	log.Debug("mounted directory is ", mountedDir)
	annotationsFilePath := path.Join(mountedDir, "metadata", "annotations.yaml")

	fileContents, err := os.ReadFile(annotationsFilePath)
	if err != nil {
		log.Error("fail to read metadata/annotation.yaml file in bundle")
		return nil, err
	}

	annotations, err := extractAnnotationsBytes(fileContents)
	if err != nil {
		log.Error("metadata/annotations.yaml found but is malformed")
		return nil, err
	}

	return annotations, nil
}

// extractAnnotationsBytes reads the annotation data read from a file and returns the expected format for that yaml
// represented as a map[string]string.
func extractAnnotationsBytes(annotationBytes []byte) (map[string]string, error) {
	type metadata struct {
		Annotations map[string]string
	}

	if len(annotationBytes) == 0 {
		return nil, errors.ErrEmptyAnnotationFile
	}

	var bundleMeta metadata
	if err := yaml.Unmarshal(annotationBytes, &bundleMeta); err != nil {
		log.Error("metadata/annotations.yaml found but is malformed")
		return nil, err
	}

	return bundleMeta.Annotations, nil
}

// annotation() accepts the annotations map and searches for the specified annotation corresponding
// with the key, which is then returned.
func annotation(annotations map[string]string, key string) (string, error) {
	log.Tracef("searching for key (%s) in bundle", key)
	log.Trace("bundle data: ", annotations)
	value, found := annotations[key]
	if !found {
		return "", fmt.Errorf("did not find value at the key %s in the annotations.yaml", key)
	}

	return value, nil
}

func getCsvFilePathFromBundle(mountedDir string) (string, error) {
	log.Trace("reading clusterserviceversion file from the bundle")
	log.Debug("mounted directory is ", mountedDir)
	matches, err := filepath.Glob(filepath.Join(mountedDir, "manifests", "*.clusterserviceversion.yaml"))

	if err != nil {
		log.Error("glob pattern is malformed: ", err)
		return "", err
	}
	if len(matches) == 0 {
		log.Error("unable to find clusterserviceversion file in the bundle image: ", err)
		return "", err
	}
	if len(matches) > 1 {
		log.Error("found more than one clusterserviceversion file in the bundle image: ", err)
		return "", err
	}
	log.Debug(fmt.Sprintf("The path to csv file is %s", matches[0]))
	return matches[0], nil
}

func getSupportedInstalledModes(mountedDir string) (map[string]bool, error) {
	csvFilepath, err := getCsvFilePathFromBundle(mountedDir)
	if err != nil {
		return nil, err
	}

	csvFileReader, err := os.ReadFile(csvFilepath)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	var csv ClusterServiceVersion
	err = yaml.Unmarshal(csvFileReader, &csv)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	var installedModes map[string]bool = make(map[string]bool, len(csv.Spec.InstallModes))
	for _, v := range csv.Spec.InstallModes {
		if v.Supported {
			installedModes[v.Type] = true
		}
	}
	return installedModes, nil
}

type ClusterServiceVersion struct {
	Spec ClusterServiceVersionSpec `yaml:"spec"`
}

type ClusterServiceVersionSpec struct {
	// InstallModes specify supported installation types
	InstallModes []InstallMode `yaml:"installModes,omitempty"`
}

type InstallMode struct {
	Type      string `yaml:"type"`
	Supported bool   `yaml:"supported"`
}
