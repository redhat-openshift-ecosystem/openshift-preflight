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
	"strings"

	"github.com/operator-framework/api/pkg/manifests"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	fileutils "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/utils/file"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/utils/migration"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

func ExtractContainerTar(tarball string) (string, error) {
	// we assume the input path is something like "abcdefg.tar", representing a container image,
	// so we need to remove the extension.
	containerIDSlice := strings.Split(filepath.Base(tarball), ".tar")
	if len(containerIDSlice) != 2 {
		// we expect a single entry in the slice, otherwise we split incorrectly
		return "", fmt.Errorf("%w: %s: %s", errors.ErrExtractingTarball, "received an improper container tarball name to extract", tarball)
	}

	outputDir := filepath.Join(filepath.Dir(tarball), containerIDSlice[0])
	err := os.Mkdir(outputDir, 0755)
	if err != nil {
		return "", fmt.Errorf("%w: %s", errors.ErrExtractingTarball, err)
	}

	_, err = exec.Command("tar", "xvf", tarball, "--directory", outputDir).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", errors.ErrExtractingTarball, err)
	}

	return outputDir, nil
}

func GetContainerFromRegistry(podmanEngine cli.PodmanEngine, imageLoc string) (string, error) {
	pullReport, err := podmanEngine.Pull(imageLoc, cli.ImagePullOptions{})
	if err != nil {
		return pullReport.StdoutErr, fmt.Errorf("%w: %s", errors.ErrGetRemoteContainerFailed, err)
	}

	return pullReport.StdoutErr, nil
}

// Calls podman to save the image to a temporary directory in /tmp/preflight-{randomString}.
// Should be followed with:
// defer os.RemoveAll(tarballDir)
// Returns the locataion of the tarball.
func SaveContainerToFilesystem(podmanEngine cli.PodmanEngine, imageLog string) (string, error) {
	inspectReport, err := podmanEngine.InspectImage(imageLog, cli.ImageInspectOptions{})
	if err != nil {
		return "", fmt.Errorf("%w: %s", errors.ErrImageInspectFailed, err)
	}

	tempdir, err := os.MkdirTemp(os.TempDir(), "preflight-*")
	if err != nil {
		return "", fmt.Errorf("%w: %s", errors.ErrCreateTempDir, err)
	}

	imgSig := inspectReport.Images[0].Id
	tarpath := filepath.Join(tempdir, imgSig+".tar")
	err = podmanEngine.Save(imgSig, []string{}, cli.ImageSaveOptions{Destination: tarpath})
	if err != nil {
		return "", fmt.Errorf("%w: %s", errors.ErrSaveContainerFailed, err)
	}
	return tarpath, nil
}

type ContainerFn func(certification.ImageReference) (bool, error)

// RunInsideImageFS executes a provided function by mounting the image filesystem
// to the host. Note that any ContainerFn that is expected to run using this function
// should know that the input is a filesystem path.
func RunInsideImageFS(podmanEngine cli.PodmanEngine, image string, containerFn ContainerFn) (bool, error) {
	report, err := podmanEngine.MountImage(image)
	if err != nil {
		log.Error("stdout: ", report.Stdout)
		log.Error("stderr: ", report.Stderr)
		log.Error("could not mount filesystem", err)
		return false, err
	}

	defer func() {
		report, err := podmanEngine.UnmountImage(image)
		if err != nil {
			log.Warn("stdout: ", report.Stdout)
			log.Warn("stderr: ", report.Stderr)
		}
	}()

	return containerFn(migration.ImageToImageReference(strings.TrimSpace(report.MountDir)))
}

func DeprecatedGenerateBundleHash(podmanEngine cli.PodmanEngine, image string) (string, error) {
	hashCmd := `find . -not -name "Dockerfile" -type f -printf '%f\t%p\n' | sort -V -k1 | cut -d$'\t' -f2 | tr '\n' '\0' | xargs -r0 -I {} md5sum "{}"` // >> $HOME/hashes.txt`
	bundleCmd := fmt.Sprintf("pushd $(podman image mount %[1]s) &> /dev/null && %[2]s && popd &> /dev/null && podman image unmount %[1]s &> /dev/null", image, hashCmd)
	report, err := podmanEngine.Unshare(map[string]string{}, "/bin/bash", "-c", bundleCmd)
	if err != nil {
		log.Errorf("could not generate bundle hash")
		log.Debugf(fmt.Sprintf("Stdout: %s", report.Stdout))
		log.Debugf(fmt.Sprintf("Stderr: %s", report.Stderr))
		return "", err
	}
	log.Tracef(fmt.Sprintf("Hash is: %s", report.Stdout))
	err = os.WriteFile(fileutils.ArtifactPath("hashes.txt"), []byte(report.Stdout), 0644)
	if err != nil {
		log.Errorf("could not write bundle hash file")
		return "", err
	}
	sum := md5.Sum([]byte(report.Stdout))

	return fmt.Sprintf("%x", sum), nil
}

func GenerateBundleHash(image string) (string, error) {
	// TODO: Convert this to regular Go commands
	hashCmd := `find . -not -name "Dockerfile" -type f -printf '%f\t%p\n' | sort -V -k1 | cut -d$'\t' -f2 | tr '\n' '\0' | xargs -r0 -I {} md5sum "{}"` // >> $HOME/hashes.txt`
	cmd := exec.Command("/bin/bash", "-c", hashCmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Errorf("could not generate bundle hash")
		log.Debugf(fmt.Sprintf("Stdout: %s", stdout.String()))
		log.Debugf(fmt.Sprintf("Stderr: %s", stderr.String()))
		return "", err
	}
	log.Tracef(fmt.Sprintf("Hash is: %s", stdout.String()))
	err = os.WriteFile(fileutils.ArtifactPath("hashes.txt"), stdout.Bytes(), 0644)
	if err != nil {
		log.Errorf("could not write bundle hash file")
		return "", err
	}
	sum := md5.Sum(stdout.Bytes())

	return fmt.Sprintf("%x", sum), nil
}

// ReadBundle will accept the manifests directory where a bundle is expected to live,
// and walks the directory to find all bundle assets.
func ReadBundle(manifestsDir string) (*operatorsv1alpha1.ClusterServiceVersion, error) {
	bundle, err := manifests.GetBundleFromDir(manifestsDir)
	if err != nil {
		return nil, err
	}

	return bundle.CSV, nil
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
