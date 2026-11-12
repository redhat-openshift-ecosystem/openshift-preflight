package pyxis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pyxis Submit", func() {
	ctx := context.Background()

	var pyxisClient *pyxisClient
	var certInput CertificationInput
	var capturedArtifacts []Artifact
	// capturedCalls records "METHOD PATH" for every request, in order, so tests can
	// assert on the sequence of API calls made during a submission.
	var capturedCalls []string
	// capturedImagePosts records the image payloads POSTed to /api/v1/images, so tests
	// can assert on what createImage actually sent.
	var capturedImagePosts []CertImage
	// capturedImagePatches records the image payloads PATCHed to /api/v1/images/id/<id>,
	// so tests can assert on what updateImage actually sent.
	var capturedImagePatches []CertImage
	mux := http.NewServeMux()

	// These go from most explicit to least explicit. They will be check that way by the ServeMux.
	mux.HandleFunc("/api/v1/projects/certification/id/my-awesome-project-id/test-results", pyxisTestResultsHandler(ctx))
	mux.HandleFunc("/api/v1/projects/certification/id/my-image-project-id/test-results", pyxisTestResultsHandler(ctx))
	mux.HandleFunc("/api/v1/projects/certification/test-results/id/54321", pyxisTestResultsHandler(ctx))
	mux.HandleFunc("/api/v1/projects/certification/id/my-image-project-id/images", pyxisImageHandler(ctx))
	mux.HandleFunc("/api/v1/projects/certification/id/", pyxisProjectHandler(ctx))
	mux.HandleFunc("/api/v1/images/id/updateImage", pyxisImageHandler(ctx))
	mux.HandleFunc("/api/v1/images/id/blah", pyxisImageHandler(ctx))
	mux.HandleFunc("/api/v1/images/id/blah/", pyxisRPMManifestHandler(ctx))
	mux.HandleFunc("/api/v1/images/id/updateImage/", pyxisImageHandler(ctx))
	mux.HandleFunc("/api/v1/images", pyxisImageHandler(ctx))

	// Wraps mux to record every request made during a test, so tests can assert
	// on what was actually sent to Pyxis instead of only the overall submission result.
	capturingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCalls = append(capturedCalls, fmt.Sprintf("%s %s", r.Method, r.URL.Path))

		if r.Body != nil {
			if body, err := io.ReadAll(r.Body); err == nil {
				r.Body = io.NopCloser(bytes.NewReader(body))

				if r.Method == http.MethodPost && r.URL.Path == "/api/v1/images" {
					var incoming CertImage
					if json.Unmarshal(body, &incoming) == nil {
						capturedImagePosts = append(capturedImagePosts, incoming)
					}
				}

				if r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/api/v1/images/id/") {
					var incoming CertImage
					if json.Unmarshal(body, &incoming) == nil {
						capturedImagePatches = append(capturedImagePatches, incoming)
					}
				}

				if r.Method == http.MethodPost && r.URL.Path == "/api/v1/projects/certification/id/my-awesome-project-id/artifacts" {
					var incoming Artifact
					if json.Unmarshal(body, &incoming) == nil {
						capturedArtifacts = append(capturedArtifacts, incoming)
					}
				}
			}
		}
		mux.ServeHTTP(w, r)
	})

	BeforeEach(func() {
		capturedArtifacts = nil
		capturedCalls = nil
		capturedImagePosts = nil
		capturedImagePatches = nil
		pyxisClient = NewPyxisClient(
			"my.pyxis.host/api",
			"my-spiffy-api-token",
			"my-awesome-project-id",
			&http.Client{Transport: localRoundTripper{handler: capturingHandler}},
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
		Context("and POST call to test-results is success and PATCH is a failure", func() {
			JustBeforeEach(func() {
				pyxisClient.APIToken = "my-bad-results-patch-api-token"
			})
			It("should get an unknown error", func() {
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
		Context("and artifacts are present", func() {
			JustBeforeEach(func() {
				certInput.Artifacts = []Artifact{
					{Filename: "artifact-one.json", Content: "{}"},
					{Filename: "artifact-two.json", Content: "{}"},
				}
			})
			It("should set ImageID on each artifact and submit successfully", func() {
				certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
				Expect(err).ToNot(HaveOccurred(), "artifact submission should not fail")
				Expect(capturedArtifacts).To(HaveLen(2), "createArtifact should be called once per artifact")
				for _, artifact := range capturedArtifacts {
					Expect(artifact.ImageID).To(Equal(certResults.CertImage.ID), "each artifact payload should carry the certImage ID")
				}
			})
		})
		Context("and artifact creation fails", func() {
			JustBeforeEach(func() {
				certInput.Artifacts = []Artifact{{Filename: "artifact.json", Content: "{}"}}
				pyxisClient.APIToken = "my-bad-artifact-api-token"
			})
			It("should return an error", func() {
				certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
				Expect(err).To(HaveOccurred(), "createArtifact failure should propagate as an error")
				Expect(certResults).To(BeNil(), "certResults should be nil when artifact creation fails")
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

		Context("createImage 409 Conflict, getImage 200, and deferred updateImage 200", func() {
			BeforeEach(func() {
				pyxisClient.APIToken = "my-update-image-success-api-token"
				pyxisClient.ProjectID = "my-image-project-id"
				certInput.CertImage.Certified = true
			})
			Context("when a project is submitted", func() {
				Context("and the image already exists but is not yet certified", func() {
					It("should finalize the certified flag via updateImage after all data is submitted", func() {
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

		Context("createImage 409 Conflict, getImage 200, and deferred updateImage 500", func() {
			BeforeEach(func() {
				pyxisClient.APIToken = "my-update-image-failure-api-token"
				pyxisClient.ProjectID = "my-image-project-id"
				certInput.CertImage.Certified = true
			})
			Context("when a project is submitted", func() {
				Context("and the deferred certification update fails", func() {
					It("should error on finalization", func() {
						certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
						Expect(err).To(HaveOccurred())
						Expect(certResults).To(BeNil())
					})
				})
			})
		})

		Context("createImage 409 Conflict and getImage returns a soft-deleted record first", func() {
			BeforeEach(func() {
				pyxisClient.APIToken = "my-deleted-image-api-token"
				pyxisClient.ProjectID = "my-image-project-id"
				certInput.CertImage.Certified = true
			})
			Context("when the digest also has an active image", func() {
				It("should submit against the active image", func() {
					certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
					Expect(err).ToNot(HaveOccurred())
					Expect(certResults).ToNot(BeNil())
					Expect(certResults.CertImage.ID).To(Equal("updateImage"),
						"results and certification must target the active image, not the deleted one")
					Expect(capturedImagePatches).To(HaveLen(1))
					Expect(capturedImagePatches[0].ID).To(Equal("updateImage"))
				})
			})
		})

		Context("createImage 409 Conflict and getImage returns a deleted record after the active one", func() {
			BeforeEach(func() {
				pyxisClient.APIToken = "my-trailing-deleted-image-api-token"
				pyxisClient.ProjectID = "my-image-project-id"
				certInput.CertImage.Certified = true
			})
			Context("when the active image is neither the first nor the last record", func() {
				It("should still submit against the active image", func() {
					certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
					Expect(err).ToNot(HaveOccurred())
					Expect(certResults).ToNot(BeNil())
					Expect(certResults.CertImage.ID).To(Equal("updateImage"))
				})
			})
		})

		Context("createImage 409 Conflict and getImage returns only deleted images", func() {
			BeforeEach(func() {
				pyxisClient.APIToken = "my-all-deleted-image-api-token"
				pyxisClient.ProjectID = "my-image-project-id"
				certInput.CertImage.Certified = true
			})
			It("should error instead of submitting against a deleted image", func() {
				certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no active image found"))
				Expect(certResults).To(BeNil())
			})
		})

		Context("createImage 409 Conflict and getImage returns no images", func() {
			BeforeEach(func() {
				pyxisClient.APIToken = "my-empty-image-api-token"
				pyxisClient.ProjectID = "my-image-project-id"
				certInput.CertImage.Certified = true
			})
			It("should error instead of panicking", func() {
				certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no image found"))
				Expect(certResults).To(BeNil())
			})
		})

		Context("deferred certification on a fresh image", func() {
			BeforeEach(func() {
				certInput.CertImage.Certified = true
			})
			Context("when a project is submitted", func() {
				It("should create the image as not certified and certify it last", func() {
					certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
					Expect(err).ToNot(HaveOccurred())
					Expect(certResults).ToNot(BeNil())

					Expect(capturedImagePosts).To(HaveLen(1), "createImage should be called exactly once")
					Expect(capturedImagePosts[0].Certified).To(BeFalse(), "the initial image POST must not carry the certified flag")

					Expect(capturedImagePatches).To(HaveLen(1), "updateImage should be called exactly once")
					Expect(capturedImagePatches[0].ID).To(Equal("blah"))
					Expect(capturedImagePatches[0].Certified).To(BeTrue(), "the final updateImage PATCH must carry the certified flag")

					Expect(certResults.CertImage.Certified).To(BeTrue(), "the final image should be certified")

					imagePatchIndex := slices.Index(capturedCalls, "PATCH /api/v1/images/id/blah")
					resultsPatchIndex := slices.Index(capturedCalls, "PATCH /api/v1/projects/certification/test-results/id/54321")
					Expect(imagePatchIndex).To(BeNumerically(">", -1), "updateImage should have been called")
					Expect(resultsPatchIndex).To(BeNumerically(">", -1), "updateTestResults should have been called")
					Expect(imagePatchIndex).To(BeNumerically(">", resultsPatchIndex),
						"the certified flag must be applied after the test results are linked to the image")

					Expect(certInput.CertImage.Certified).To(BeTrue(),
						"SubmitResults must not clear the caller's certified flag")
					Expect(certInput.CertProject.CertificationStatus).To(Equal("Started"),
						"SubmitResults must not mutate the caller's project")
					Expect(certInput.RpmManifest.ImageID).To(BeEmpty(),
						"SubmitResults must not link the image ID on the caller's rpm manifest")
				})
			})

			Context("when the final updateImage fails", func() {
				JustBeforeEach(func() {
					pyxisClient.APIToken = "my-finalize-failure-api-token"
					pyxisClient.ProjectID = "my-image-project-id"
				})
				It("should error with guidance to re-run when the image was not previously certified", func() {
					certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("could not update image certification status"))
					Expect(err.Error()).To(ContainSubstring("re-running preflight check container with --submit will complete the submission"))
					Expect(certResults).To(BeNil())
				})
			})
		})

		Context("deferred certification when checks did not pass", func() {
			JustBeforeEach(func() {
				pyxisClient.APIToken = "my-finalize-failure-api-token"
				pyxisClient.ProjectID = "my-image-project-id"
			})
			Context("when a project is submitted", func() {
				It("should not call updateImage at all", func() {
					certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
					Expect(err).ToNot(HaveOccurred(), "no PATCH to the image should be attempted when the image is not certified")
					Expect(certResults).ToNot(BeNil())
					Expect(certResults.CertImage.Certified).To(BeFalse())
					Expect(slices.Index(capturedCalls, "PATCH /api/v1/images/id/blah")).To(Equal(-1),
						"updateImage must not be called when the image is not certified")
				})
			})
		})

		Context("createImage 409 Conflict for an already certified image", func() {
			BeforeEach(func() {
				// The token would make any PATCH fail, so this proves the update is skipped.
				pyxisClient.APIToken = "my-certified-image-500-api-token"
				pyxisClient.ProjectID = "my-image-project-id"
				certInput.CertImage.Certified = true
			})
			Context("when a project is submitted", func() {
				It("should skip updateImage and succeed", func() {
					certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
					Expect(err).ToNot(HaveOccurred())
					Expect(certResults).ToNot(BeNil())
					Expect(certResults.CertImage.Certified).To(BeTrue())
					Expect(slices.Index(capturedCalls, "PATCH /api/v1/images/id/blah")).To(Equal(-1),
						"an already certified image must not be patched, since Pyxis rejects updates to published images")
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

	Context("createTestResults 400 version not supported", func() {
		BeforeEach(func() {
			pyxisClient.APIToken = "my-unsupported-preflight-version"
			pyxisClient.ProjectID = "my-awesome-project-id"
		})
		Context("when a project is submitted", func() {
			Context("and an unsupported preflight version is sent to createTestResults", func() {
				It("should error", func() {
					certResults, err := pyxisClient.SubmitResults(ctx, &certInput)
					Expect(err).To(HaveOccurred())
					Expect(certResults).To(BeNil())
				})
			})
		})
	})

	Context("createTestResults 400 version not recognized", func() {
		BeforeEach(func() {
			pyxisClient.APIToken = "my-unrecognized-preflight-version"
			pyxisClient.ProjectID = "my-awesome-project-id"
		})
		Context("when a project is submitted", func() {
			Context("and an unrecognized preflight version is sent to createTestResults", func() {
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
