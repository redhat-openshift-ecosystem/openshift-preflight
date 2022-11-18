package artifacts_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/artifacts"
)

var _ = Describe("Artifacts package context management", func() {
	Context("When working with an ArtifactWriter from context", func() {
		It("Should be settable and retrievable using helper functions", func() {
			aw, err := artifacts.NewMapWriter()
			Expect(err).ToNot(HaveOccurred())

			ctx := artifacts.ContextWithWriter(context.Background(), aw)
			awRetrieved := artifacts.WriterFromContext(ctx)
			Expect(awRetrieved).ToNot(BeNil())
			Expect(awRetrieved).To(BeEquivalentTo(aw))
		})
	})
	It("Should return nil when there is no ArtifactWriter found in the context", func() {
		awRetrieved := artifacts.WriterFromContext(context.Background())
		Expect(awRetrieved).To(BeNil())
	})
})
