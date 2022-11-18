package cmd

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("runtime-assets test", func() {
	Context("When formatting JSON data", func() {
		It("should be formatted in a standard format", func() {
			in := map[string]interface{}{
				"foo":  "bar",
				"this": []string{"that", "theother"},
			}
			expected := "{\n    \"foo\": \"bar\",\n    \"this\": [\n        \"that\",\n        \"theother\"\n    ]\n}"
			res, err := prettyPrintJSON(in)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(expected))
		})

		Context("With invalid data", func() {
			It("should throw an error", func() {
				// channels are not supported in json.
				_, err := prettyPrintJSON(make(chan int))
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("When printing the runtime assets", func() {
		It("should print successfully and match the actual data", func() {
			buf := bytes.NewBuffer([]byte{})
			err := printAssets(context.TODO(), buf)
			Expect(err).ToNot(HaveOccurred())

			var printed runtime.AssetData
			Expect(json.Unmarshal(buf.Bytes(), &printed)).To(Succeed())

			actual := runtime.Assets(context.TODO())

			Expect(printed).To(BeEquivalentTo(actual))
		})
	})

	Context("When calling the runtime-assets cobra command", func() {
		BeforeEach(createAndCleanupDirForArtifactsAndLogs)
		It("should print successfully and match the actual data", func() {
			out, err := executeCommand(runtimeAssetsCmd())
			Expect(err).ToNot(HaveOccurred())

			var printed runtime.AssetData
			Expect(json.Unmarshal([]byte(out), &printed)).To(Succeed())

			actual := runtime.Assets(context.TODO())

			Expect(printed).To(BeEquivalentTo(actual))
		})
	})
})
