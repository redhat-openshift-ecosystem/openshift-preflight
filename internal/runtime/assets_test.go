package runtime

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"net/url"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Runtime assets tests", func() {
	var src, srcHost string
	var digest v1.Hash
	BeforeEach(func() {
		// Set up a fake registry.
		registryLogger := log.New(io.Discard, "", log.Ldate)
		s := httptest.NewServer(registry.New(registry.Logger(registryLogger)))
		DeferCleanup(func() {
			s.Close()
		})
		u, err := url.Parse(s.URL)
		Expect(err).ToNot(HaveOccurred())
		src = fmt.Sprintf("%s/test/preflight", u.Host)
		srcHost = u.Hostname()

		img, err := random.Image(1024, 5)
		Expect(err).ToNot(HaveOccurred())

		digest, err = img.Digest()
		Expect(err).ToNot(HaveOccurred())

		err = crane.Push(img, src)
		Expect(err).ToNot(HaveOccurred())

		images["assettest"] = fmt.Sprintf("%s:latest", src)
	})
	Context("when asking for assets", func() {
		Context("the registry works", func() {
			It("should return a digest list", func() {
				data := Assets(context.TODO())
				Expect(data.Images).To(ContainElement(fmt.Sprintf("%s@%s", srcHost, digest.String())))
			})
		})
		Context("the registry throws an error", func() {
			It("should not return the image", func() {
				data := Assets(context.TODO())
				Expect(data.Images).ToNot(ContainElement("quay.io/operator-framework/scorecard-test@sha256:deadb33f"))
			})
		})
	})
})

var _ = Describe("Scorecard Image tests", func() {
	Context("when getting the Scorecard image", func() {
		Context("the default is used", func() {
			It("should return the default", func() {
				image := ScorecardImage(context.Background(), "")
				Expect(image).To(Equal(images["scorecard"]))
			})
		})
		Context("the image is overidden", func() {
			It("should return the passed param", func() {
				image := ScorecardImage(context.Background(), "quay.io/some/container:v1.0.0")
				Expect(image).To(Equal("quay.io/some/container:v1.0.0"))
			})
		})
	})
})
