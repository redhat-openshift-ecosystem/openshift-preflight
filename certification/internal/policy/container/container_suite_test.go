package container

import (
	"errors"
	"io"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
)

func TestContainer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Container Suite")
}

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.TraceLevel)
}

type FakeLayer struct{}

func (fl FakeLayer) Digest() (v1.Hash, error) {
	return v1.Hash{}, nil
}

func (fl FakeLayer) DiffID() (v1.Hash, error) {
	return v1.Hash{}, nil
}

func (fl FakeLayer) Compressed() (io.ReadCloser, error) {
	return nil, nil
}

func (fl FakeLayer) Uncompressed() (io.ReadCloser, error) {
	return nil, nil
}

func (fl FakeLayer) Size() (int64, error) {
	return 0, nil
}

func (fl FakeLayer) MediaType() (types.MediaType, error) {
	return "mediatype", nil
}

type FakeSkopeoEngine struct {
	SkopeoReportStdout string
	SkopeoReportStderr string
	Tags               []string
}

type SkopeoData struct {
	Repository string
	Tags       []string
}

func (fse FakeSkopeoEngine) ListTags(image string) (*cli.SkopeoListTagsReport, error) {
	skopeoReport := cli.SkopeoListTagsReport{
		Stdout: fse.SkopeoReportStdout,
		Stderr: fse.SkopeoReportStderr,
		Tags:   fse.Tags,
	}
	return &skopeoReport, nil
}

type BadSkopeoEngine struct{}

func (bse BadSkopeoEngine) ListTags(string) (*cli.SkopeoListTagsReport, error) {
	skopeoReport := cli.SkopeoListTagsReport{
		Stdout: "Bad Stdout",
		Stderr: "Bad stderr",
		Tags:   []string{""},
	}
	return &skopeoReport, errors.New("the Skopeo ListTags has failed")
}
