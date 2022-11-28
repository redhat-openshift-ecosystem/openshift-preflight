package pyxis

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pyxis", func() {
	ctx := context.Background()
	mux := http.NewServeMux()
	mux.HandleFunc("/query/", pyxisGraphqlFindImagesHandler(ctx))
	pyxisClient := NewPyxisClient("my.pyxis.host/query/", "my-spiffy-api-token", "my-awesome-project-id", &http.Client{Transport: localRoundTripper{handler: mux}})

	Context("Find Images", func() {
		Context("and one image is passed", func() {
			It("should return one image", func() {
				certImages, err := pyxisClient.FindImagesByDigest(ctx, []string{"sha256:deadb33f"})
				Expect(err).ToNot(HaveOccurred())
				Expect(certImages).ToNot(BeNil())
				Expect(certImages).ToNot(BeZero())
				Expect(certImages[0].Certified).To(BeTrue())
			})
		})
		Context("and an error occurs", func() {
			It("should return nil and an error", func() {
				errorMux := http.NewServeMux()
				errorMux.Handle("/query/", &errorHandler{})
				pyxisClient.Client = &http.Client{Transport: localRoundTripper{handler: errorMux}}
				certImages, err := pyxisClient.FindImagesByDigest(ctx, []string{"sha256:dontmatter"})
				Expect(err).To(HaveOccurred())
				Expect(certImages).To(BeNil())
			})
			AfterEach(func() {
				pyxisClient.Client = &http.Client{Transport: localRoundTripper{handler: mux}}
			})
		})
	})
})
