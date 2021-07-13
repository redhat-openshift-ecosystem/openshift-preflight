package shell

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
)

type SkopeoCLIEngine struct{}

func (e SkopeoCLIEngine) ListTags(image string) (*cli.SkopeoListTagsReport, error) {
	imageName := strings.Split(image, ":")[0]
	cmd := exec.Command("skopeo", "list-tags", "docker://"+imageName)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		return nil, err
	}

	var report cli.SkopeoListTagsReport

	err = json.Unmarshal(stdout.Bytes(), &report)
	if err != nil {
		log.Error("unable to parse skopeo list-tags data for image", err)
		log.Debug("error marshaling skopeo list-tags data: ", err)
		log.Trace("failure in attempt to convert the raw bytes from `skopeo list-tags` to a [map[string]interface{}")
		return nil, err
	}
	report.Stdout = stdout.String()
	report.Stderr = stderr.String()

	return &report, nil
}
