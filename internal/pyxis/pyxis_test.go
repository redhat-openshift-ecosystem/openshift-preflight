package pyxis

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pyxis", func() {
	ctx := context.Background()
	mux := http.NewServeMux()
	mux.HandleFunc("/query/", pyxisGraphqlFindImagesHandler(ctx))
	mux.HandleFunc("/api/v1/projects/certification/test-results/id/54321", pyxisTestResultsHandler(ctx))
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

	Context("createArtifact", func() {
		It("should return the created artifact on success", func() {
			artifactMux := http.NewServeMux()
			artifactMux.HandleFunc("/api/v1/projects/certification/id/test-project-id/artifacts", func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal(http.MethodPost))
				defer r.Body.Close()
				body, err := io.ReadAll(r.Body)
				Expect(err).ToNot(HaveOccurred())

				var incoming Artifact
				Expect(json.Unmarshal(body, &incoming)).To(Succeed())

				resp := Artifact{
					ID:          "artifact-123",
					CertProject: incoming.CertProject,
					Content:     incoming.Content,
					ContentType: incoming.ContentType,
					FileSize:    incoming.FileSize,
					Filename:    incoming.Filename,
					ImageID:     incoming.ImageID,
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				Expect(json.NewEncoder(w).Encode(resp)).To(Succeed())
			})

			client := NewPyxisClient(
				"my.pyxis.host/api",
				"my-spiffy-api-token",
				"test-project-id",
				&http.Client{Transport: localRoundTripper{handler: artifactMux}},
			)

			input := &Artifact{
				CertProject: "test-project-id",
				Content:     "dGVzdC1jb250ZW50",
				ContentType: "application/json",
				FileSize:    42,
				Filename:    "test-artifact.json",
				ImageID:     "image-abc",
			}

			result, err := client.createArtifact(ctx, input)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
			Expect(result.ID).To(Equal("artifact-123"))
			Expect(result.Filename).To(Equal("test-artifact.json"))
			Expect(result.CertProject).To(Equal("test-project-id"))
			Expect(result.FileSize).To(Equal(int64(42)))
		})
	})
})
