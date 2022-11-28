package pyxis

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pyxis Submit", func() {
	ctx := context.Background()

	var pyxisClient *pyxisClient
	var certInput CertificationInput
	mux := http.NewServeMux()

	// These go from most explicit to least explicit. They will be check that way by the ServeMux.
	mux.HandleFunc("/api/v1/projects/certification/id/my-awesome-project-id/test-results", pyxisTestResultsHandler(ctx))
	mux.HandleFunc("/api/v1/projects/certification/id/my-image-project-id/images", pyxisImageHandler(ctx))
	mux.HandleFunc("/api/v1/projects/certification/id/", pyxisProjectHandler(ctx))
	mux.HandleFunc("/api/v1/images/id/updateImage", pyxisImageHandler(ctx))
	mux.HandleFunc("/api/v1/images/id/blah/", pyxisRPMManifestHandler(ctx))
	mux.HandleFunc("/api/v1/images/id/updateImage/", pyxisImageHandler(ctx))
	mux.HandleFunc("/api/v1/images", pyxisImageHandler(ctx))

	BeforeEach(func() {
		pyxisClient = NewPyxisClient(
			"my.pyxis.host/api",
			"my-spiffy-api-token",
			"my-awesome-project-id",
			&http.Client{Transport: localRoundTripper{handler: mux}},
		)
		certInput = CertificationInput{
			CertProject: &CertProject{CertificationStatus: "Started"},
			CertImage: &CertImage{
				Repositories: []Repository{
					{
						Registry:   "my.registry",
						Repository: "my/repo",
					},
				},
				DockerImageDigest: "sha256:deadb33f",
			},
			RpmManifest: &RPMManifest{},
			TestResults: &TestResults{},
			Artifacts:   []Artifact{},
		}
	})

	Context("when a project is submitted", func() {
		Context("and it is not already In Progress", func() {
			It("should switch to In Progress", func() {
				certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
				Expect(err).ToNot(HaveOccurred())
				Expect(certResults).ToNot(BeNil())
				Expect(certResults.CertProject).ToNot(BeNil())
				Expect(certResults.CertImage).ToNot(BeNil())
				Expect(certResults.TestResults).ToNot(BeNil())
			})
		})
		Context("and the certImage does not have repositories", func() {
			JustBeforeEach(func() {
				certInput.CertImage = &CertImage{
					Repositories: []Repository{},
				}
			})
			It("should throw an error", func() {
				certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
				Expect(err).To(HaveOccurred())
				Expect(certResults).To(BeNil())
			})
		})
		Context("and a server error occurs", func() {
			JustBeforeEach(func() {
				pyxisClient.APIToken = "my-error-project-api-token"
			})
			It("should handle the project update error", func() {
				certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
				Expect(err).To(HaveOccurred())
				Expect(certResults).To(BeNil())
			})
		})
		Context("and the client sends a bad token", func() {
			JustBeforeEach(func() {
				pyxisClient.APIToken = "my-bad-project-api-token"
			})
			It("should get an unauthorized", func() {
				certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
				Expect(err).To(HaveOccurred())
				Expect(certResults).To(BeNil())
			})
		})
		Context("and is submitted with an empty registry", func() {
			JustBeforeEach(func() {
				certInput.CertImage = &CertImage{}
			})
			It("should get invalid cert image error", func() {
				certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
				Expect(err).To(HaveOccurred())
				Expect(certResults).To(BeNil())
			})
		})
	})

	Context("when an index.docker.io project is submitted", func() {
		BeforeEach(func() {
			pyxisClient.APIToken = "my-index-docker-io-project-api-token"
			certInput.CertImage.Repositories[0] = Repository{Registry: "index.docker.io", Repository: "my/repo"}
		})
		Context("and it is not already In Progress", func() {
			It("should switch to In Progress and certResults.CertProject.Container.Registry should equal 'docker.io'", func() {
				certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
				Expect(err).ToNot(HaveOccurred())
				Expect(certResults).ToNot(BeNil())
				Expect(certResults.CertProject.Container.Registry).Should(Equal(defaultRegistryAlias))
				Expect(certResults.CertImage.Repositories[0].Registry).Should(Equal(defaultRegistryAlias))
				Expect(certResults.TestResults).ToNot(BeNil())
			})
		})
	})

	Context("Image", func() {
		Context("createImage 409 Conflict", func() {
			BeforeEach(func() {
				pyxisClient.APIToken = "my-image-409-api-token"
				pyxisClient.ProjectID = "my-image-project-id"
			})
			Context("when a project is submitted", func() {
				Context("and the image already exists", func() {
					It("should get a conflict and handle it", func() {
						certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
						Expect(err).ToNot(HaveOccurred())
						Expect(certResults).ToNot(BeNil())
						Expect(certResults.CertProject).ToNot(BeNil())
						Expect(certResults.CertImage).ToNot(BeNil())
						Expect(certResults.TestResults).ToNot(BeNil())
					})
				})
			})
		})

		Context("createImage 401 Unauthorized", func() {
			BeforeEach(func() {
				pyxisClient.APIToken = "my-bad-image-api-token"
				pyxisClient.ProjectID = "my-image-project-id"
			})
			Context("when a project is submitted", func() {
				Context("and the api token is invalid", func() {
					It("should get an unauthorized result", func() {
						certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
						Expect(err).To(HaveOccurred())
						Expect(certResults).To(BeNil())
					})
				})
			})
		})

		Context("createImage 409 Conflict and getImage 401 Unauthorized", func() {
			BeforeEach(func() {
				pyxisClient.APIToken = "my-bad-401-image-api-token"
				pyxisClient.ProjectID = "my-image-project-id"
			})
			Context("when a project is submitted", func() {
				Context("and a bad token is sent to getImage and createImage is in conflict", func() {
					It("should error", func() {
						certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
						Expect(err).To(HaveOccurred())
						Expect(certResults).To(BeNil())
					})
				})
			})
		})

		Context("createImage 409 Conflict, getImage 200, and updateImage 200", func() {
			BeforeEach(func() {
				pyxisClient.APIToken = "my-update-image-success-api-token"
				pyxisClient.ProjectID = "my-image-project-id"
				certInput.CertImage.Certified = true
			})
			Context("when a project is submitted", func() {
				Context("and an update token is sent to getImage and createImage is in conflict", func() {
					It("should call updateImage and certified flag should be updated", func() {
						certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
						Expect(err).ToNot(HaveOccurred())
						Expect(certResults).ToNot(BeNil())
						Expect(certResults.CertProject).ToNot(BeNil())
						Expect(certResults.CertImage).ToNot(BeNil())
						Expect(certResults.CertImage.Certified).To(Equal(true))
						Expect(certResults.TestResults).ToNot(BeNil())
					})
				})
			})
		})

		Context("createImage 409 Conflict, getImage 200, and updateImage 500", func() {
			BeforeEach(func() {
				pyxisClient.APIToken = "my-update-image-failure-api-token"
				pyxisClient.ProjectID = "my-image-project-id"
				certInput.CertImage.Certified = true
			})
			Context("when a project is submitted", func() {
				Context("and an update token is sent to getImage and createImage is in conflict", func() {
					It("should call updateImage and error", func() {
						certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
						Expect(err).To(HaveOccurred())
						Expect(certResults).To(BeNil())
					})
				})
			})
		})

		Context("createImage 500 InternalServerError", func() {
			BeforeEach(func() {
				pyxisClient.APIToken = "my-bad-500-image-api-token"
				pyxisClient.ProjectID = "my-image-project-id"
			})
			Context("when a project is submitted", func() {
				Context("and an unknown error occurs", func() {
					It("should error", func() {
						certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
						Expect(err).To(HaveOccurred())
						Expect(certResults).To(BeNil())
					})
				})
			})
		})
	})

	Context("RPMManifest", func() {
		Context("createRPMManifest 409 Conflict", func() {
			BeforeEach(func() {
				pyxisClient.APIToken = "my-bad-rpmmanifest-409-api-token"
			})
			Context("when a project is submitted", func() {
				Context("and the RPM manifest already exists", func() {
					It("should retry and return success", func() {
						certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
						Expect(err).ToNot(HaveOccurred())
						Expect(certResults).ToNot(BeNil())
						Expect(certResults.CertProject).ToNot(BeNil())
						Expect(certResults.CertImage).ToNot(BeNil())
						Expect(certResults.TestResults).ToNot(BeNil())
					})
				})
			})
		})

		Context("createRPMManifest 401 Unauthorized", func() {
			BeforeEach(func() {
				pyxisClient.APIToken = "my-bad-rpmmanifest-api-token"
			})
			Context("when a project is submitted", func() {
				Context("and a bad token is sent to createRPMManifest", func() {
					It("should error", func() {
						certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
						Expect(err).To(HaveOccurred())
						Expect(certResults).To(BeNil())
					})
				})
			})
		})

		Context("createRPMManifest 409 Conflict and getRPMManifest 401 Unauthorized", func() {
			BeforeEach(func() {
				pyxisClient.APIToken = "my-bad-rpmmanifest-401-api-token"
			})
			Context("when a project is submitted", func() {
				Context("and a bad token is sent to getRPMManifest and createRPMManifest is in conflict", func() {
					It("should error", func() {
						certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
						Expect(err).To(HaveOccurred())
						Expect(certResults).To(BeNil())
					})
				})
			})
		})
	})

	Context("createTestResults 401 Unauthorized", func() {
		BeforeEach(func() {
			pyxisClient.APIToken = "my-bad-testresults-api-token"
			pyxisClient.ProjectID = "my-awesome-project-id"
		})
		Context("when a project is submitted", func() {
			Context("and a bad api token is sent to createTestResults", func() {
				It("should error", func() {
					certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
					Expect(err).To(HaveOccurred())
					Expect(certResults).To(BeNil())
				})
			})
		})
	})

	Context("GetProject", func() {
		Context("when a project is submitted", func() {
			Context("and it is not already In Progress", func() {
				It("should switch to In Progress", func() {
					certProject, err := pyxisClient.GetProject(context.Background())
					Expect(err).ToNot(HaveOccurred())
					Expect(certProject).ToNot(BeNil())
				})
			})
		})
	})

	Context("GetProject 401 Unauthorized", func() {
		BeforeEach(func() {
			pyxisClient.APIToken = "my-401-project-api-token"
		})
		Context("when trying to retrieve a project", func() {
			Context("and the API token is bad", func() {
				It("should get an unauthorized response", func() {
					certProject, err := pyxisClient.GetProject(context.Background())
					Expect(err).To(HaveOccurred())
					Expect(certProject).To(BeNil())
				})
			})
		})
	})
})
