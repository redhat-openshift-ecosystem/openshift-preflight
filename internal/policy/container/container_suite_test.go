package container

import (
	"io"
	"testing"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	. "github.com/onsi/ginkgo/v2"
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

var AssertMetaData = func(check certification.Check) {
	Context("When checking metadata", func() {
		Context("The check name should not be empty", func() {
			Expect(check.Name()).ToNot(BeEmpty())
		})

		Context("The metadata keys should not be empty", func() {
			meta := check.Metadata()
			Expect(meta.CheckURL).ToNot(BeEmpty())
			Expect(meta.Description).ToNot(BeEmpty())
			Expect(meta.KnowledgeBaseURL).ToNot(BeEmpty())
			// Level is optional.
		})

		Context("The help text should not be empty", func() {
			help := check.Help()
			Expect(help.Message).ToNot(BeEmpty())
			Expect(help.Suggestion).ToNot(BeEmpty())
		})
	})
}
