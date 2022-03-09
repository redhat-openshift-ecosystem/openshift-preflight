package pyxis

import (
	"context"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var ctx = context.Background()

var _ = Describe("Pyxis Submit", func() {
	var pyxisEngine *pyxisEngine

	BeforeEach(func() {
		pyxisEngine = NewPyxisEngine("my-spiffy-api-token", "my-awseome-project-id", fakeHttpClient{})
	})
	Context("when a project is submitted", func() {
		Context("and it is not already In Progress", func() {
			It("should switch to In Progress", func() {
				certProject, certImage, testResults, err := pyxisEngine.SubmitResults(ctx, &CertProject{CertificationStatus: "Started"}, &CertImage{}, &RPMManifest{}, &TestResults{})
				Expect(err).ToNot(HaveOccurred())
				Expect(certProject).ToNot(BeNil())
				Expect(certImage).ToNot(BeNil())
				Expect(testResults).ToNot(BeNil())
			})
		})
	})
})

var _ = Describe("Pyxis Submit updateProject 401 Unauthorized", func() {
	var pyxisEngine *pyxisEngine

	BeforeEach(func() {
		pyxisEngine = NewPyxisEngine("my-spiffy-api-token", "my-awseome-project-id", fakeHttpCertProjectUnauthorizedClient{})
	})
	Context("when a project is submitted", func() {
		Context("and it is not already In Progress", func() {
			It("should switch to In Progress", func() {
				certProject, certImage, testResults, err := pyxisEngine.SubmitResults(ctx, &CertProject{CertificationStatus: "Started"}, &CertImage{}, &RPMManifest{}, &TestResults{})
				Expect(err).To(MatchError(fmt.Errorf("%w: %s", errors.New("error calling remote API"), "could not retrieve project")))
				Expect(certProject).To(BeNil())
				Expect(certImage).To(BeNil())
				Expect(testResults).To(BeNil())
			})
		})
	})
})

var _ = Describe("Pyxis Submit with createImage 409 Conflict", func() {
	var pyxisEngine *pyxisEngine

	BeforeEach(func() {
		pyxisEngine = NewPyxisEngine("my-spiffy-api-token", "my-awseome-project-id", fakeHttpCreateImageConflictClient{})
	})
	Context("when a project is submitted", func() {
		Context("and it is not already In Progress", func() {
			It("should switch to In Progress", func() {
				certProject, certImage, testResults, err := pyxisEngine.SubmitResults(ctx, &CertProject{}, &CertImage{}, &RPMManifest{}, &TestResults{})
				Expect(err).ToNot(HaveOccurred())
				Expect(certProject).ToNot(BeNil())
				Expect(certImage).ToNot(BeNil())
				Expect(testResults).ToNot(BeNil())
			})
		})
	})
})

var _ = Describe("Pyxis Submit with createImage 401 Unauthorized", func() {
	var pyxisEngine *pyxisEngine

	BeforeEach(func() {
		pyxisEngine = NewPyxisEngine("my-spiffy-api-token", "my-awseome-project-id", fakeHttpCreateImageUnauthorizedClient{})
	})
	Context("when a project is submitted", func() {
		Context("and it is not already In Progress", func() {
			It("should switch to In Progress", func() {
				certProject, certImage, testResults, err := pyxisEngine.SubmitResults(ctx, &CertProject{CertificationStatus: "Started"}, &CertImage{}, &RPMManifest{}, &TestResults{})
				Expect(err).To(MatchError(errors.New("error calling remote API")))
				Expect(certProject).To(BeNil())
				Expect(certImage).To(BeNil())
				Expect(testResults).To(BeNil())
			})
		})
	})
})

var _ = Describe("Pyxis Submit with createImage 409 Conflict and getImage 401 Unauthorized ", func() {
	var pyxisEngine *pyxisEngine

	BeforeEach(func() {
		pyxisEngine = NewPyxisEngine("my-spiffy-api-token", "my-awseome-project-id", fakeHttpCreateImageConflictAndUnauthorizedClient{})
	})
	Context("when a project is submitted", func() {
		Context("and it is not already In Progress", func() {
			It("should switch to In Progress", func() {
				certProject, certImage, testResults, err := pyxisEngine.SubmitResults(ctx, &CertProject{CertificationStatus: "Started"}, &CertImage{}, &RPMManifest{}, &TestResults{})
				Expect(err).To(MatchError(errors.New("error calling remote API")))
				Expect(certProject).To(BeNil())
				Expect(certImage).To(BeNil())
				Expect(testResults).To(BeNil())
			})
		})
	})
})

var _ = Describe("Pyxis Submit with createRPMManifest 409 Conflict", func() {
	var pyxisEngine *pyxisEngine

	BeforeEach(func() {
		pyxisEngine = NewPyxisEngine("my-spiffy-api-token", "my-awseome-project-id", fakeHttpCreateRPMManifestConflictClient{})
	})
	Context("when a project is submitted", func() {
		Context("and it is not already In Progress", func() {
			It("should switch to In Progress", func() {
				certProject, certImage, testResults, err := pyxisEngine.SubmitResults(ctx, &CertProject{CertificationStatus: "Started"}, &CertImage{}, &RPMManifest{}, &TestResults{})
				Expect(err).ToNot(HaveOccurred())
				Expect(certProject).ToNot(BeNil())
				Expect(certImage).ToNot(BeNil())
				Expect(testResults).ToNot(BeNil())
			})
		})
	})
})

var _ = Describe("Pyxis Submit with createRPMManifest 401 Unauthorized", func() {
	var pyxisEngine *pyxisEngine

	BeforeEach(func() {
		pyxisEngine = NewPyxisEngine("my-spiffy-api-token", "my-awseome-project-id", fakeHttpCreateRPMManifestUnauthorizedClient{})
	})
	Context("when a project is submitted", func() {
		Context("and it is not already In Progress", func() {
			It("should switch to In Progress", func() {
				certProject, certImage, testResults, err := pyxisEngine.SubmitResults(ctx, &CertProject{CertificationStatus: "Started"}, &CertImage{}, &RPMManifest{}, &TestResults{})
				Expect(err).To(MatchError(errors.New("error calling remote API")))
				Expect(certProject).To(BeNil())
				Expect(certImage).To(BeNil())
				Expect(testResults).To(BeNil())
			})
		})
	})
})

var _ = Describe("Pyxis Submit with createRPMManifest 409 Conflict and getRPMManifest 401 Unauthorized", func() {
	var pyxisEngine *pyxisEngine

	BeforeEach(func() {
		pyxisEngine = NewPyxisEngine("my-spiffy-api-token", "my-awseome-project-id", fakeHttpCreateRPMManifestConflictAndUnauthorizedClient{})
	})
	Context("when a project is submitted", func() {
		Context("and it is not already In Progress", func() {
			It("should switch to In Progress", func() {
				certProject, certImage, testResults, err := pyxisEngine.SubmitResults(ctx, &CertProject{CertificationStatus: "Started"}, &CertImage{}, &RPMManifest{}, &TestResults{})
				Expect(err).To(MatchError(errors.New("error calling remote API")))
				Expect(certProject).To(BeNil())
				Expect(certImage).To(BeNil())
				Expect(testResults).To(BeNil())
			})
		})
	})
})

var _ = Describe("Pyxis Submit with createTestResults 401 Unauthorized", func() {
	var pyxisEngine *pyxisEngine

	BeforeEach(func() {
		pyxisEngine = NewPyxisEngine("my-spiffy-api-token", "my-awseome-project-id", fakeHttpCreateTestResultsUnauthorizedClient{})
	})
	Context("when a project is submitted", func() {
		Context("and it is not already In Progress", func() {
			It("should switch to In Progress", func() {
				certProject, certImage, testResults, err := pyxisEngine.SubmitResults(ctx, &CertProject{CertificationStatus: "Started"}, &CertImage{}, &RPMManifest{}, &TestResults{})
				Expect(err).To(MatchError(errors.New("error calling remote API")))
				Expect(certProject).To(BeNil())
				Expect(certImage).To(BeNil())
				Expect(testResults).To(BeNil())
			})
		})
	})
})

var _ = Describe("Pyxis GetProejct", func() {
	var pyxisEngine *pyxisEngine

	BeforeEach(func() {
		pyxisEngine = NewPyxisEngine("my-spiffy-api-token", "my-awseome-project-id", fakeHttpClient{})
	})
	Context("when a project is submitted", func() {
		Context("and it is not already In Progress", func() {
			It("should switch to In Progress", func() {
				certProject, err := pyxisEngine.GetProject(context.Background())
				Expect(err).ToNot(HaveOccurred())
				Expect(certProject).ToNot(BeNil())
			})
		})
	})
})

var _ = Describe("Pyxis GetProject 401 Unauthorized", func() {
	var pyxisEngine *pyxisEngine

	BeforeEach(func() {
		pyxisEngine = NewPyxisEngine("my-spiffy-api-token", "my-awseome-project-id", fakeHttpCertProjectUnauthorizedClient{})
	})
	Context("when a project is submitted", func() {
		Context("and it is not already In Progress", func() {
			It("should switch to In Progress", func() {
				certProject, err := pyxisEngine.GetProject(context.Background())
				Expect(err).To(MatchError(errors.New("error calling remote API")))
				Expect(certProject).To(BeNil())
			})
		})
	})
})
