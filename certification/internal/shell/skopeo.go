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
	var skopeoData map[string]interface{}

	err = json.Unmarshal(stdout.Bytes(), &skopeoData)
	if err != nil {
		log.Error("unable to parse skopeo list-tags data for image", err)
		log.Debug("error marshaling skopeo list-tags data: ", err)
		log.Trace("failure in attempt to convert the raw bytes from `skopeo list-tags` to a [map[string]interface{}")
		return nil, err
	}
	jsonData := skopeoData["Tags"].([]interface{})

	var tags []string = make([]string, len(jsonData))

	for i, tag := range jsonData {
		tags[i] = tag.(string)
	}

	return &cli.SkopeoListTagsReport{Stdout: stdout.String(), Tags: tags, Stderr: stderr.String()}, nil
}
