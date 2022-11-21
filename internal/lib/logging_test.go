package lib

import (
	"context"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"
)

var _ = Describe("Lib Logging Functions", func() {
	Context("When logging using the library logging helpers", func() {
		DescribeTable("The library helpers should successfully detect if the context is our CLI",
			func(ctx context.Context, expected bool) {
				actual := CallerIsCLI(ctx)
				Expect(expected).To(Equal(actual))
			},
			Entry("With the context being configured using the library's setting method", SetCallerToCLI(context.Background()), true),
			Entry("With the context not containing the CLI context Key", context.Background(), false),
			Entry("With the context being manually configured to contain the key with a value of false", context.WithValue(context.Background(), executionEnvIsCLI, false), false),
		)

		It("Should emit logs to the writer configure in context, if configured", func() {
			w, err := artifacts.NewMapWriter()
			Expect(err).ToNot(HaveOccurred())
			ctx := artifacts.ContextWithWriter(context.Background(), w)

			LogThroughArtifactWriterIfSet(ctx)
			msg := "testing logs emitted through artifact writer"
			log.L().Info(msg)
			data, ok := w.Files()["preflight.log"]
			Expect(ok).To(BeTrue())
			Expect(data).To(ContainSubstring(msg))
		})

		It("Should be configured to discard logs if no artifact writer is configured", func() {
			ctx := context.Background()
			LogThroughArtifactWriterIfSet(ctx)
			Expect(log.L().Out).To(Equal(io.Discard))
		})
	})
})
