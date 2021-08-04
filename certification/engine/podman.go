package engine

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

func NewPodmanEngine() *cli.PodmanEngine {
	var engine cli.PodmanEngine = podmanEngine{}
	return &engine
}

type podmanEngine struct{}

func (pe podmanEngine) Pull(rawImage string, opts cli.ImagePullOptions) (*cli.ImagePullReport, error) {
	stdouterr, err := exec.Command("podman", "pull", rawImage).CombinedOutput()
	if err != nil {
		return &cli.ImagePullReport{StdoutErr: string(stdouterr)}, err
	}

	return &cli.ImagePullReport{StdoutErr: string(stdouterr)}, nil
}

func (pe podmanEngine) Run(opts cli.ImageRunOptions) (*cli.ImageRunReport, error) {
	cmdArgs := []string{"run", "-it", "--rm", "--log-level", opts.LogLevel, "--entrypoint", opts.EntryPoint, opts.Image}
	cmdArgs = append(cmdArgs, opts.EntryPointArgs...)
	cmd := exec.Command("podman", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return &cli.ImageRunReport{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: cmd.ProcessState.ExitCode()}, err
	}
	return &cli.ImageRunReport{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: cmd.ProcessState.ExitCode()}, nil
}

func (pe podmanEngine) Save(nameOrID string, tags []string, opts cli.ImageSaveOptions) error {
	cmdArgs := []string{"save", "--output", opts.Destination}
	cmdArgs = append(cmdArgs, nameOrID)
	_, err := exec.Command("podman", cmdArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrSaveContainerFailed, err)
	}
	return nil
}

func (pe podmanEngine) InspectImage(rawImage string, opts cli.ImageInspectOptions) (*cli.ImageInspectReport, error) {
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
func (pe podmanEngine) ScanImage(image string) (*cli.ImageScanReport, error) {
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
func (pe podmanEngine) Create(rawImage string, opts *cli.PodmanCreateOptions) (*cli.PodmanCreateReport, error) {
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
func (pe podmanEngine) CopyFrom(containerID, sourcePath, destinationPath string) (*cli.PodmanCopyReport, error) {
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
func (pe podmanEngine) Remove(containerID string) (*cli.PodmanRemoveReport, error) {
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

// Mount will attempt to mount the filesystem of the container at containerID, and returns the mounted path.
func (pe podmanEngine) Mount(containerId string) (*cli.PodmanMountReport, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("podman", "mount", containerId)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Error("could not run mount")
		return &cli.PodmanMountReport{Stdout: stdout.String(), Stderr: stderr.String()}, err
	}
	mountedDir := strings.TrimSpace(stdout.String())
	return &cli.PodmanMountReport{MountDir: mountedDir, Stdout: stdout.String(), Stderr: stderr.String()}, nil
}

func (pe podmanEngine) Unmount(containerId string) (*cli.PodmanUnmountReport, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("podman", "unmount", containerId)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	report := &cli.PodmanUnmountReport{Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		log.Errorf("could not run unmount. Output: %s", stderr.String())
		return report, err
	}
	return report, nil
}

// MountImage will attempt to mount a filesystem of the image at imageID, and returns the mounted path.
func (pe podmanEngine) MountImage(imageID string) (*cli.PodmanMountReport, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("podman", "image", "mount", imageID)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Error("could not run image mount")
		return &cli.PodmanMountReport{Stdout: stdout.String(), Stderr: stderr.String()}, err
	}
	mountedDir := strings.TrimSpace(stdout.String())
	return &cli.PodmanMountReport{MountDir: mountedDir, Stdout: stdout.String(), Stderr: stderr.String()}, nil
}

// UnmountImage will attempt to unmount a filesystem of the image at imageID.
func (pe podmanEngine) UnmountImage(imageID string) (*cli.PodmanUnmountReport, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("podman", "image", "unmount", imageID)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	report := &cli.PodmanUnmountReport{Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		log.Errorf("could not run image unmount. Output: %s", stderr.String())
		return report, err
	}
	return report, nil
}

func (pe podmanEngine) Unshare(env map[string]string, command ...string) (*cli.PodmanUnshareReport, error) {
	var stdout, stderr bytes.Buffer
	cmdLine := []string{"unshare"}
	cmdLine = append(cmdLine, command...)
	cmd := exec.Command("podman", cmdLine...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if _, ok := os.LookupEnv("PREFLIGHT_EXEC_RUN"); ok {
		// We've already been called. This should not already be set. Unwind.
		return &cli.PodmanUnshareReport{Stdout: stdout.String(), Stderr: stderr.String()}, errors.ErrAlreadyInUnshare
	}

	loglevel := os.Getenv("PFLT_LOGLEVEL")

	environ := []string{
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		"PREFLIGHT_EXEC_RUN=1",
		"PFLT_LOGFILE=preflight-unshare.log",
		fmt.Sprintf("PFLT_LOGLEVEL=%s", loglevel),
	}

	for k, v := range env {
		environ = append(environ, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = environ

	err := cmd.Run()
	if err != nil {
		log.Error("could not run command in unshare")
		return &cli.PodmanUnshareReport{Stdout: stdout.String(), Stderr: stderr.String()}, err
	}

	return &cli.PodmanUnshareReport{Stdout: stdout.String(), Stderr: stderr.String()}, err
}

func (pe podmanEngine) UnshareWithCheck(check, image string, mounted bool) (*cli.PodmanUnshareCheckReport, error) {
	env := map[string]string{
		"PATH":                 os.Getenv("PATH"),
		"PREFLIGHT_EXEC_CHECK": check,
		"PREFLIGHT_EXEC_IMAGE": image,
	}

	if mounted {
		env["PREFLIGHT_EXEC_MOUNTED"] = fmt.Sprintf("%t", mounted)
	}

	unshareReport, err := pe.Unshare(env, os.Args[0], "check", "run")
	if err != nil {
		return &cli.PodmanUnshareCheckReport{PodmanUnshareReport: *unshareReport, PassedOverall: false}, err
	}

	var results cli.PodmanUnshareCheckReport
	err = json.Unmarshal([]byte(unshareReport.Stdout), &results)
	if err != nil {
		log.Error("could not read results from stdout")
		return &cli.PodmanUnshareCheckReport{PodmanUnshareReport: *unshareReport, PassedOverall: false}, err
	}
	return &results, nil
}
