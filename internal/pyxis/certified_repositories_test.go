package pyxis

import (
	"context"
	"net/http"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pyxis CheckRedHatRepositories", func() {
	ctx := context.Background()
	var pyxisClient *pyxisClient
	mux := http.NewServeMux()
	mux.HandleFunc("/query/", pyxisGraphqlCertifiedRepositoriesHandler(ctx))

	Context("when some layers are provided", func() {
		BeforeEach(func() {
			pyxisClient = NewPyxisClient("my.pyxis.host/query/", "my-spiffy-api-token", "my-awesome-project-id", &http.Client{Transport: localRoundTripper{handler: mux}})
		})

		Context("and a layer is associated with a certified repository", func() {
			It("should return the repository metadata", func() {
				certRepositories, err := pyxisClient.CertifiedRepositoriesForLayers(ctx, []cranev1.Hash{{}})
				Expect(err).ToNot(HaveOccurred(), "CertifiedRepositoriesForLayers should not return an error")
				Expect(certRepositories).ToNot(BeNil(), "certRepositories should not be nil")
				Expect(certRepositories).To(HaveLen(1), "certRepositories should contain the certified repository")
				Expect(certRepositories[0].ID).To(Equal("repo-1"))
				Expect(certRepositories[0].Registry).To(Equal(accessRegistry))
				Expect(certRepositories[0].Repository).To(Equal("ubi9/ubi"))
				Expect(certRepositories[0].EOLDate).To(BeNil())
				Expect(certRepositories[0].ReleaseCategories).To(Equal([]string{"Generally Available"}))
			})
		})
		Context("and the repository query returns an error", func() {
			BeforeEach(func() {
				errorMux := http.NewServeMux()
				errorMux.HandleFunc("/query/", pyxisGraphqlCertifiedRepositoriesErrorHandler(ctx))
				pyxisClient.Client = &http.Client{Transport: localRoundTripper{handler: errorMux}}
			})

			It("should return an error", func() {
				certRepositories, err := pyxisClient.CertifiedRepositoriesForLayers(ctx, []cranev1.Hash{{}})
				Expect(certRepositories).To(BeNil())
				Expect(err).To(MatchError("repository query returned an error for image image-1 and repository ubi9/ubi: status 404: repository not found"))
			})
		})
	})
})

func certifiedRepositoriesClient(response string) *pyxisClient {
	mux := http.NewServeMux()
	mux.HandleFunc("/query/", func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			defer r.Body.Close()
		}
		w.Header().Set("Content-Type", "application/json")
		mustWrite(w, response)
	})

	return NewPyxisClient("my.pyxis.host/query/", "my-spiffy-api-token", "my-awesome-project-id", &http.Client{Transport: localRoundTripper{handler: mux}})
}

var _ = Describe("CertifiedRepositoriesForLayers response handling", func() {
	ctx := context.Background()

	It("returns a find_images error", func() {
		client := certifiedRepositoriesClient(`{"data":{"find_images":{"error":{"status":400,"detail":"query failed"},"data":[]}}}`)

		repositories, err := client.CertifiedRepositoriesForLayers(ctx, []cranev1.Hash{{}})
		Expect(repositories).To(BeNil())
		Expect(err).To(MatchError("cert image query returned an error: status 400: query failed"))
	})

	It("ignores repositories from other registries", func() {
		client := certifiedRepositoriesClient(`{"data":{"find_images":{"error":null,"data":[{"_id":"image-1","repositories":[{"registry":"quay.io","repository":"ubi9/ubi","edges":null}]}]}}}`)

		repositories, err := client.CertifiedRepositoriesForLayers(ctx, []cranev1.Hash{{}})
		Expect(err).ToNot(HaveOccurred())
		Expect(repositories).To(BeEmpty())
	})

	It("returns an error when repository edge data is null", func() {
		client := certifiedRepositoriesClient(`{"data":{"find_images":{"error":null,"data":[{"_id":"image-1","repositories":[{"registry":"registry.access.redhat.com","repository":"ubi9/ubi","edges":null}]}]}}}`)

		repositories, err := client.CertifiedRepositoriesForLayers(ctx, []cranev1.Hash{{}})
		Expect(repositories).To(BeNil())
		Expect(err).To(MatchError("repository data for image image-1 and repository ubi9/ubi were null"))
	})

	It("returns an error when container repository data is null", func() {
		client := certifiedRepositoriesClient(`{"data":{"find_images":{"error":null,"data":[{"_id":"image-1","repositories":[{"registry":"registry.access.redhat.com","repository":"ubi9/ubi","edges":{"repository":{"error":null,"data":null}}}]}]}}}`)

		repositories, err := client.CertifiedRepositoriesForLayers(ctx, []cranev1.Hash{{}})
		Expect(repositories).To(BeNil())
		Expect(err).To(MatchError("container repository for image image-1 and repository ubi9/ubi were null"))
	})

	It("returns valid repositories when another repository edge has an error", func() {
		client := certifiedRepositoriesClient(`{"data":{"find_images":{"error":null,"data":[{"_id":"image-1","repositories":[{"registry":"registry.access.redhat.com","repository":"broken","edges":{"repository":{"error":{"status":404,"detail":"repository not found"},"data":null}}},{"registry":"registry.access.redhat.com","repository":"ubi9/ubi","edges":{"repository":{"error":null,"data":{"_id":"repo-1","registry":"registry.access.redhat.com","repository":"ubi9/ubi","eol_date":null,"release_categories":["Generally Available"]}}}}]}]}}}`)

		repositories, err := client.CertifiedRepositoriesForLayers(ctx, []cranev1.Hash{{}})
		Expect(err).ToNot(HaveOccurred())
		Expect(repositories).To(HaveLen(1))
		Expect(repositories[0].ID).To(Equal("repo-1"))
	})

	It("parses a repository EOL date", func() {
		client := certifiedRepositoriesClient(`{"data":{"find_images":{"error":null,"data":[{"_id":"image-1","repositories":[{"registry":"registry.access.redhat.com","repository":"ubi9/ubi","edges":{"repository":{"error":null,"data":{"_id":"repo-1","registry":"registry.access.redhat.com","repository":"ubi9/ubi","eol_date":"2026-01-15T12:00:00Z","release_categories":["Generally Available"]}}}}]}]}}}`)

		repositories, err := client.CertifiedRepositoriesForLayers(ctx, []cranev1.Hash{{}})
		Expect(err).ToNot(HaveOccurred())
		Expect(repositories).To(HaveLen(1))
		Expect(repositories[0].EOLDate).ToNot(BeNil())
	})

	It("returns an error for an invalid repository EOL date", func() {
		client := certifiedRepositoriesClient(`{"data":{"find_images":{"error":null,"data":[{"_id":"image-1","repositories":[{"registry":"registry.access.redhat.com","repository":"ubi9/ubi","edges":{"repository":{"error":null,"data":{"_id":"repo-1","registry":"registry.access.redhat.com","repository":"ubi9/ubi","eol_date":"not-a-date","release_categories":["Generally Available"]}}}}]}]}}}`)

		repositories, err := client.CertifiedRepositoriesForLayers(ctx, []cranev1.Hash{{}})
		Expect(repositories).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`invalid RFC3339 eol_date "not-a-date"`))
	})
})
