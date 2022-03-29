package pyxis

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var ctx = context.Background()

var _ = Describe("Pyxis Submit", func() {
	var pyxisClient *pyxisClient
	mux := http.NewServeMux()
	mux.Handle("/api/v1/projects/certification/id/", &pyxisProjectHandler{})
	mux.Handle("/api/v1/images", &pyxisImageHandler{})
	mux.Handle("/api/v1/images/id/blah/", &pyxisRPMManifestHandler{})
	mux.Handle("/api/v1/projects/certification/id/my-awesome-project-id/test-results", &pyxisTestResultsHandler{})

	Context("when a project is submitted", func() {
		BeforeEach(func() {
			pyxisClient = NewPyxisClient("my.pyxis.host/api", "my-spiffy-api-token", "my-awesome-project-id", &http.Client{Transport: localRoundTripper{handler: mux}})
		})
		Context("and it is not already In Progress", func() {
			It("should switch to In Progress", func() {
				certResults, err := pyxisClient.SubmitResults(ctx, &CertificationInput{
					CertProject: &CertProject{CertificationStatus: "Started"},
					CertImage: &CertImage{
						Repositories: []Repository{
							{
								Registry:   "my.registry",
								Repository: "my/repo",
							},
						},
					},
					RpmManifest: &RPMManifest{},
					TestResults: &TestResults{},
					Artifacts:   []Artifact{},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(certResults).ToNot(BeNil())
				Expect(certResults.CertProject).ToNot(BeNil())
				Expect(certResults.CertImage).ToNot(BeNil())
				Expect(certResults.TestResults).ToNot(BeNil())
			})
		})
	})

	Context("updateProject 401 Unauthorized", func() {
		BeforeEach(func() {
			pyxisClient = NewPyxisClient("my.pyxis.host/api", "my-bad-project-api-token", "my-awesome-project-id", &http.Client{Transport: localRoundTripper{handler: mux}})
		})
		Context("when a project is submitted", func() {
			Context("and the client sends a bad token", func() {
				It("should get an unauthorized", func() {
					certResults, err := pyxisClient.SubmitResults(ctx, &CertificationInput{
						CertProject: &CertProject{CertificationStatus: "Started"},
						CertImage: &CertImage{
							Repositories: []Repository{
								{
									Registry:   "my.registry",
									Repository: "my/repo",
								},
							},
						},
						RpmManifest: &RPMManifest{},
						TestResults: &TestResults{},
						Artifacts:   []Artifact{},
					})
					Expect(err).To(MatchError(fmt.Errorf("%w: %s", errors.New("error calling remote API"), "could not retrieve project")))
					Expect(certResults).To(BeNil())
				})
			})
		})
	})

	Context("createImage 409 Conflict", func() {
		BeforeEach(func() {
			pyxisClient = NewPyxisClient("my.pyxis.host/api", "my-spiffy-api-token", "my-image-409-project-id", &http.Client{Transport: localRoundTripper{handler: mux}})
		})
		Context("when a project is submitted", func() {
			Context("and the image already exists", func() {
				It("should get a conflict and handle it", func() {
					certResults, err := pyxisClient.SubmitResults(ctx, &CertificationInput{
						CertProject: &CertProject{},
						CertImage: &CertImage{
							Repositories: []Repository{
								{
									Registry:   "my.registry",
									Repository: "my/repo",
								},
							},
						},
						RpmManifest: &RPMManifest{},
						TestResults: &TestResults{},
						Artifacts:   []Artifact{},
					})
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
			pyxisClient = NewPyxisClient("my.pyxis.host/api", "my-bad-image-api-token", "my-awesome-project-id", &http.Client{Transport: localRoundTripper{handler: mux}})
		})
		Context("when a project is submitted", func() {
			Context("and the api token is invalid", func() {
				It("should get an unauthorized result", func() {
					certResults, err := pyxisClient.SubmitResults(ctx, &CertificationInput{
						CertProject: &CertProject{CertificationStatus: "Started"},
						CertImage: &CertImage{
							Repositories: []Repository{
								{
									Registry:   "my.registry",
									Repository: "my/repo",
								},
							},
						},
						RpmManifest: &RPMManifest{},
						TestResults: &TestResults{},
						Artifacts:   []Artifact{},
					})
					Expect(err).To(MatchError(errors.New("error calling remote API")))
					Expect(certResults).To(BeNil())
				})
			})
		})
	})

	Context("createImage 409 Conflict and getImage 401 Unauthorized", func() {
		BeforeEach(func() {
			pyxisClient = NewPyxisClient("my.pyxis.host/api", "my-bad-image-api-token", "my-image-409-project-id", &http.Client{Transport: localRoundTripper{handler: mux}})
		})
		Context("when a project is submitted", func() {
			Context("and a bad token is sent to getImage and createImage is in conflict", func() {
				It("should error", func() {
					certResults, err := pyxisClient.SubmitResults(ctx, &CertificationInput{
						CertProject: &CertProject{CertificationStatus: "Started"},
						CertImage: &CertImage{
							Repositories: []Repository{
								{
									Registry:   "my.registry",
									Repository: "my/repo",
								},
							},
						},
						RpmManifest: &RPMManifest{},
						TestResults: &TestResults{},
						Artifacts:   []Artifact{},
					})
					Expect(err).To(MatchError(errors.New("error calling remote API")))
					Expect(certResults).To(BeNil())
				})
			})
		})
	})

	Context("createRPMManifest 409 Conflict", func() {
		BeforeEach(func() {
			pyxisClient = NewPyxisClient("my.pyxis.host/api", "my-spiffy-api-token", "my-awesome-project-id", &http.Client{Transport: localRoundTripper{handler: mux}})
		})
		Context("when a project is submitted", func() {
			Context("and the RPM manifest already exists", func() {
				It("should retry and return success", func() {
					certResults, err := pyxisClient.SubmitResults(ctx, &CertificationInput{
						CertProject: &CertProject{},
						CertImage: &CertImage{
							Repositories: []Repository{
								{
									Registry:   "my.registry",
									Repository: "my/repo",
								},
							},
						},
						RpmManifest: &RPMManifest{},
						TestResults: &TestResults{},
						Artifacts:   []Artifact{},
					})
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
			pyxisClient = NewPyxisClient("my.pyxis.host/api", "my-bad-rpmmanifest-api-token", "my-awesome-project-id", &http.Client{Transport: localRoundTripper{handler: mux}})
		})
		Context("when a project is submitted", func() {
			Context("and a bad token is sent to createRPMManifest", func() {
				It("should error", func() {
					certResults, err := pyxisClient.SubmitResults(ctx, &CertificationInput{
						CertProject: &CertProject{CertificationStatus: "Started"},
						CertImage: &CertImage{
							Repositories: []Repository{
								{
									Registry:   "my.registry",
									Repository: "my/repo",
								},
							},
						},
						RpmManifest: &RPMManifest{},
						TestResults: &TestResults{},
						Artifacts:   []Artifact{},
					})
					Expect(err).To(MatchError(errors.New("error calling remote API")))
					Expect(certResults).To(BeNil())
				})
			})
		})
	})

	Context("createRPMManifest 409 Conflict and getRPMManifest 401 Unauthorized", func() {
		BeforeEach(func() {
			pyxisClient = NewPyxisClient("my.pyxis.host/api", "my-bad-rpmmanifest-api-token", "my-manifest-409-project-id", &http.Client{Transport: localRoundTripper{handler: mux}})
		})
		Context("when a project is submitted", func() {
			Context("and a bad token is sent to getRPMManifest and createRPMManifest is in conflict", func() {
				It("should error", func() {
					certResults, err := pyxisClient.SubmitResults(ctx, &CertificationInput{
						CertProject: &CertProject{CertificationStatus: "Started"},
						CertImage: &CertImage{
							Repositories: []Repository{
								{
									Registry:   "my.registry",
									Repository: "my/repo",
								},
							},
						},
						RpmManifest: &RPMManifest{},
						TestResults: &TestResults{},
						Artifacts:   []Artifact{},
					})
					Expect(err).To(MatchError(errors.New("error calling remote API")))
					Expect(certResults).To(BeNil())
				})
			})
		})
	})

	Context("createTestResults 401 Unauthorized", func() {
		BeforeEach(func() {
			pyxisClient = NewPyxisClient("my.pyxis.host/api", "my-bad-testresults-api-token", "my-awesome-project-id", &http.Client{Transport: localRoundTripper{handler: mux}})
		})
		Context("when a project is submitted", func() {
			Context("and a bad api token is sent to createTestResults", func() {
				It("should error", func() {
					certResults, err := pyxisClient.SubmitResults(ctx, &CertificationInput{
						CertProject: &CertProject{CertificationStatus: "Started"},
						CertImage: &CertImage{
							Repositories: []Repository{
								{
									Registry:   "my.registry",
									Repository: "my/repo",
								},
							},
						},
						RpmManifest: &RPMManifest{},
						TestResults: &TestResults{},
						Artifacts:   []Artifact{},
					})
					Expect(err).To(MatchError(errors.New("error calling remote API")))
					Expect(certResults).To(BeNil())
				})
			})
		})
	})

	Context("GetProject", func() {
		BeforeEach(func() {
			pyxisClient = NewPyxisClient("my.pyxis.host/api", "my-spiffy-api-token", "my-awesome-project-id", &http.Client{Transport: localRoundTripper{handler: mux}})
		})
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
			pyxisClient = NewPyxisClient("my.pyxis.host/api", "my-bad-project-api-token", "my-awesome-project-id", &http.Client{Transport: localRoundTripper{handler: mux}})
		})
		Context("when trying to retrieve a project", func() {
			Context("and the API token is bad", func() {
				It("should get an unauthorized response", func() {
					certProject, err := pyxisClient.GetProject(context.Background())
					Expect(err).To(MatchError(errors.New("error calling remote API")))
					Expect(certProject).To(BeNil())
				})
			})
		})
	})
})
