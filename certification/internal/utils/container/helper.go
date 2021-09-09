package container

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	certutils "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/utils"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

func GenerateBundleHash(image string) (string, error) {
	// TODO: Convert this to regular Go commands
	hashCmd := fmt.Sprintf(`cd %s && find . -not -name "Dockerfile" -type f -printf '%%f\t%%p\n' | sort -V -k1 | cut -d$'\t' -f2 | tr '\n' '\0' | xargs -r0 -I {} md5sum "{}"`, image) // >> $HOME/hashes.txt`

	cmd := exec.Command("/bin/bash", "-c", hashCmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Error("could not generate bundle hash")
		log.Debugf(fmt.Sprintf("Stdout: %s", stdout.String()))
		log.Debugf(fmt.Sprintf("Stderr: %s", stderr.String()))
		return "", err
	}

	log.Tracef(fmt.Sprintf("Hash is: %s", stdout.String()))
	err = os.WriteFile(filepath.Join(certutils.ArtifactPath(), "hashes.txt"), stdout.Bytes(), 0644)
	if err != nil {
		log.Error("could not write bundle hash file")
		return "", err
	}

	sum := md5.Sum(stdout.Bytes())

	log.Debugf("md5 sum: %s", fmt.Sprintf("%x", sum))

	return fmt.Sprintf("%x", sum), nil
}

func GetAnnotationsFromBundle(mountedDir string) (map[string]string, error) {
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

// Annotation() accepts the annotations map and searches for the specified annotation corresponding
// with the key, which is then returned.
func Annotation(annotations map[string]string, key string) (string, error) {
	log.Tracef("searching for key (%s) in bundle", key)
	log.Trace("bundle data: ", annotations)
	value, found := annotations[key]
	if !found {
		return "", fmt.Errorf("did not find value at the key %s in the annotations.yaml", key)
	}

	return value, nil
}

func ImageName(image string) (string, error) {
	re, err := regexp.Compile(`(?P<Image>[^@:]+)[@|:]+.*`)

	if err != nil {
		log.Error("unable to parse the image name: ", err)
		log.Debug("error parsing the image name: ", err)
		log.Trace("failure in attempt to parse the image name to strip tag/digest")
		return "", err
	}
	if len(re.FindStringSubmatch(image)) != 0 {
		return re.FindStringSubmatch(image)[1], nil
	}

	return "", errors.ErrInvalidImageName
}

type metadata struct {
	Annotations map[string]string
}
