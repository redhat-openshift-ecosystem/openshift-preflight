package container

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	certutils "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/utils"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

func GenerateBundleHash(bundlePath string) (string, error) {
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

	err := os.WriteFile(filepath.Join(certutils.ArtifactPath(), "hashes.txt"), hashBuffer.Bytes(), 0644)
	if err != nil {
		return "", err
	}

	sum := fmt.Sprintf("%x", md5.Sum(hashBuffer.Bytes()))

	log.Debugf("md5 sum: %s", sum)

	return sum, nil
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
