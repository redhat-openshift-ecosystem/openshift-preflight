package container

import (
	"errors"
	"io"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
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

type GoodPodmanEngine struct{}

func (pe GoodPodmanEngine) PullImage(imageURI string, options cli.ImagePullOptions) (*cli.PodmanOutput, error) {
	return &cli.PodmanOutput{}, nil
}

func (pe GoodPodmanEngine) CreateContainer(imageURI string, createOptions cli.PodmanCreateOption) (*cli.PodmanCreateOutput, error) {
	return &cli.PodmanCreateOutput{
		ContainerId: "containerId",
	}, nil
}

func (p GoodPodmanEngine) StartContainer(nameOrId string) (*cli.PodmanOutput, error) {
	return &cli.PodmanOutput{}, nil
}

func (p GoodPodmanEngine) RemoveContainer(containerId string) error {
	return nil
}

func (p GoodPodmanEngine) WaitContainer(containerId string, waitOptions cli.WaitOptions) (bool, error) {
	return true, nil
}

type BadPodmanEngine struct{}

func (pe BadPodmanEngine) PullImage(imageURI string, options cli.ImagePullOptions) (*cli.PodmanOutput, error) {
	return &cli.PodmanOutput{}, nil
}

func (pe BadPodmanEngine) CreateContainer(imageURI string, createOptions cli.PodmanCreateOption) (*cli.PodmanCreateOutput, error) {
	return &cli.PodmanCreateOutput{
		ContainerId: "containerId",
	}, nil
}

func (p BadPodmanEngine) StartContainer(nameOrId string) (*cli.PodmanOutput, error) {
	return &cli.PodmanOutput{}, nil
}

func (p BadPodmanEngine) RemoveContainer(containerId string) error {
	return nil
}

func (p BadPodmanEngine) WaitContainer(containerId string, waitOptions cli.WaitOptions) (bool, error) {
	return false, errors.New("the container wait had failed")
}
