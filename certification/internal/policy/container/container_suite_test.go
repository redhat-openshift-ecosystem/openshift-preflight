package container

import (
	"io"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
