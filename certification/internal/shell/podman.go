package shell

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/utils/file"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
)

const (
	ovalFilename = "rhel-8.oval.xml.bz2"
	ovalUrl      = "https://www.redhat.com/security/data/oval/v2/RHEL8/"
	reportFile   = "vuln.html"
)

type PodmanCLIEngine struct{}

func (pe PodmanCLIEngine) Pull(rawImage string, opts cli.ImagePullOptions) (*cli.ImagePullReport, error) {
	stdouterr, err := exec.Command("podman", "pull", rawImage).CombinedOutput()
	if err != nil {
		return &cli.ImagePullReport{StdoutErr: string(stdouterr)}, err
	}

	return &cli.ImagePullReport{StdoutErr: string(stdouterr)}, nil
}

func (pe PodmanCLIEngine) Run(opts cli.ImageRunOptions) (*cli.ImageRunReport, error) {
	cmdArgs := []string{"run", "-it", "--rm", "--log-level", opts.LogLevel, "--entrypoint", opts.EntryPoint, opts.Image}
	cmdArgs = append(cmdArgs, opts.EntryPointArgs...)
	cmd := exec.Command("podman", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return &cli.ImageRunReport{Stdout: stdout.String(), Stderr: stderr.String()}, err
	}
	return &cli.ImageRunReport{Stdout: stdout.String(), Stderr: stderr.String()}, nil
}

func (pe PodmanCLIEngine) Save(nameOrID string, tags []string, opts cli.ImageSaveOptions) error {
	cmdArgs := []string{"save", "--output", opts.Destination}
	cmdArgs = append(cmdArgs, nameOrID)
	_, err := exec.Command("podman", cmdArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrSaveContainerFailed, err)
	}
	return nil
}

func (pe PodmanCLIEngine) InspectImage(rawImage string, opts cli.ImageInspectOptions) (*cli.ImageInspectReport, error) {
	cmdArgs := []string{"image", "inspect"}
	if opts.LogLevel != "" {
		cmdArgs = append(cmdArgs, "--log-level", opts.LogLevel)
	}
	cmdArgs = append(cmdArgs, rawImage)

	cmd := exec.Command("podman", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrImageInspectFailed, err)
	}

	var inspectData []cli.PodmanImage
	err = json.Unmarshal(stdout.Bytes(), &inspectData)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrImageInspectFailed, err)
	}
	return &cli.ImageInspectReport{Images: inspectData}, nil
}

// ScanImage takes the `image` and runs `oscap-podman` against the image.
// It also saves results to the vuln.html file in the current directory,
// for the end users' reference.
func (pe PodmanCLIEngine) ScanImage(image string) (*cli.ImageScanReport, error) {
	ovalFileUrl := fmt.Sprintf("%s%s", ovalUrl, ovalFilename)
	dir, err := ioutil.TempDir(".", "oval-")

	if err != nil {
		log.Error("Unable to create temp dir", err)
		return nil, err
	}
	log.Debugf("Oval file dir: %s", dir)
	defer os.RemoveAll(dir)

	ovalFilePath := filepath.Join(dir, ovalFilename)
	log.Debugf("Oval file path: %s", ovalFilePath)

	err = file.DownloadFile(ovalFilePath, ovalFileUrl)
	if err != nil {
		log.Error("Unable to download Oval file", err)
		return nil, err
	}
	// get the file name
	r := regexp.MustCompile(`(?P<filename>.*).bz2`)
	ovalFilePathDecompressed := filepath.Join(dir, r.FindStringSubmatch(ovalFilename)[1])

	err = file.Unzip(ovalFilePath, ovalFilePathDecompressed)
	if err != nil {
		log.Error("Unable to unzip Oval file: ", err)
		return nil, err
	}

	// run oscap-podman command and save the report to vuln.html
	cmd := exec.Command("oscap-podman", image, "oval", "eval", "--report", reportFile, ovalFilePathDecompressed)
	var out bytes.Buffer
	cmd.Stdout = &out
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()

	if err != nil {
		log.Error("unable to execute oscap-podman on the image: ", cmd.Stderr, reportFile)
		return nil, err
	}
	// get the current directory
	path, err := os.Getwd()
	if err != nil {
		log.Error("unable to get the current directory: ", err)
	}

	log.Debugf("The path to vulnerability report: %s/%s ", path, reportFile)

	return &cli.ImageScanReport{Stdout: out.String(), Stderr: stderr.String()}, nil
}

// Create simply creates a stopped container from the image provided and returns the container ID.
// It is the responsibility of the caller to clean up the image after use.
func (pe PodmanCLIEngine) Create(rawImage string, opts *cli.PodmanCreateOptions) (*cli.PodmanCreateReport, error) {
	cmdArgs := []string{"create"}

	if opts.Entrypoint != "" {
		cmdArgs = append(cmdArgs, "--entrypoint", opts.Entrypoint)
	}

	cmdArgs = append(cmdArgs, rawImage)

	cmd := exec.Command("podman", cmdArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Trace("Running podman with the following invocation", cmdArgs)
	err := cmd.Run()
	if err != nil {
		return &cli.PodmanCreateReport{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, nil
	}

	return &cli.PodmanCreateReport{
		Stdout:      stdout.String(),
		Stderr:      stderr.String(),
		ContainerID: strings.TrimSpace(stdout.String()),
	}, nil
}

// CopyFrom will copy the sourcePath from the container at the specified containerID to the destinationPath.
func (pe PodmanCLIEngine) CopyFrom(containerID, sourcePath, destinationPath string) (*cli.PodmanCopyReport, error) {
	cmdArgs := []string{"cp"}
	sourceArg := strings.Join([]string{containerID, sourcePath}, ":")

	cmdArgs = append(cmdArgs, sourceArg, destinationPath)
	cmd := exec.Command("podman", cmdArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Trace("Running podman with the following invocation", cmdArgs)
	err := cmd.Run()
	if err != nil {
		return &cli.PodmanCopyReport{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, err
	}

	return &cli.PodmanCopyReport{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}, nil
}

// Remove will attempt to remove a given container from the system.
func (pe PodmanCLIEngine) Remove(containerID string) (*cli.PodmanRemoveReport, error) {
	cmdArgs := []string{"rm"}
	cmdArgs = append(cmdArgs, containerID)

	cmd := exec.Command("podman", cmdArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	log.Trace("Running podman with the following invocation", cmdArgs)

	err := cmd.Run()
	if err != nil {
		return &cli.PodmanRemoveReport{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, err
	}

	return &cli.PodmanRemoveReport{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}, nil
}
