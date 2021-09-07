package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	containerutil "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/utils/container"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
)

func NewSkopeoEngine() *cli.SkopeoEngine {
	var engine cli.SkopeoEngine = SkopeoEngine{}
	return &engine
}

type SkopeoEngine struct{}

func (e SkopeoEngine) ListTags(image string) (*cli.SkopeoListTagsReport, error) {

	imageName, err := containerutil.ImageName(image)

	if err != nil {
		log.Error("unable to parse the image name", err)
		log.Debug("error parsing the image name: ", err)
		log.Trace("failure in attempt to parse the image name to strip tag/digest")
		return nil, err
	}

	cmdArgs := []string{"list-tags", "docker://" + imageName}

	log.Trace("running skopeo with the following invocation", cmdArgs)
	cmd := exec.Command("skopeo", cmdArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	if err != nil {
		return &cli.SkopeoListTagsReport{Stdout: stdout.String(), Stderr: stderr.String()}, err
	}

	var report cli.SkopeoListTagsReport

	err = json.Unmarshal(stdout.Bytes(), &report)
	if err != nil {
		log.Error("unable to parse skopeo list-tags data for image: ", err)
		log.Debug("error marshaling skopeo list-tags data: ", err)
		log.Trace("failure in attempt to convert the raw bytes from `skopeo list-tags` to a [map[string]interface{}")
		return nil, err
	}
	report.Stdout = stdout.String()
	report.Stderr = stderr.String()

	log.Debugf(fmt.Sprintf("detected these tags for %s: %s", imageName, report.Tags))

	return &report, nil
}

// InspectImage will use skopeo to inspect the input image given input options.
func (e SkopeoEngine) InspectImage(image string, opts cli.SkopeoInspectOptions) (*cli.SkopeoInspectReport, error) {
	cmdArgs := []string{"inspect"}

	// add options to the command string
	if opts.Raw {
		cmdArgs = append(cmdArgs, "--raw")
	}

	cmdArgs = append(cmdArgs, "docker://"+image)

	log.Trace("running skopeo with the following invocation", cmdArgs)
	cmd := exec.Command("skopeo", cmdArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		return &cli.SkopeoInspectReport{Stdout: stdout.String(), Stderr: stderr.String()}, err
	}

	return &cli.SkopeoInspectReport{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Blob:   stdout.Bytes(),
	}, nil
}
