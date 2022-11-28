package pyxis

import (
	"context"
	"net/http"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pyxis CheckRedHatLayers", func() {
	ctx := context.Background()
	var pyxisClient *pyxisClient
	mux := http.NewServeMux()
	mux.HandleFunc("/query/", pyxisGraphqlLayerHandler(ctx))

	Context("when some layers are provided", func() {
		BeforeEach(func() {
			pyxisClient = NewPyxisClient("my.pyxis.host/query/", "my-spiffy-api-token", "my-awesome-project-id", &http.Client{Transport: localRoundTripper{handler: mux}})
		})
		Context("and a layer is a known good layer", func() {
			It("should be a good layer", func() {
				certImages, err := pyxisClient.CertifiedImagesContainingLayers(ctx, []cranev1.Hash{{}})
				Expect(err).ToNot(HaveOccurred())
				Expect(certImages).ToNot(BeNil())
				Expect(certImages).ToNot(BeZero())
			})
		})
	})
})
