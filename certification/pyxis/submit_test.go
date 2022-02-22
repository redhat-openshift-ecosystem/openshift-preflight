package pyxis

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pyxis Submit", func() {
	var pyxisEngine *pyxisEngine

	BeforeEach(func() {
		pyxisEngine = NewPyxisEngine("my-spiffy-api-token", "my-awseome-project-id", fakeHttpClient{})
	})
	Context("when a project is submitted", func() {
		Context("and it is not already In Progress", func() {
			It("should switch to In Progress", func() {
				certProject, certImage, testResults, err := pyxisEngine.SubmitResults(&CertProject{}, &CertImage{}, &RPMManifest{}, &TestResults{})
				Expect(err).ToNot(HaveOccurred())
				Expect(certProject).ToNot(BeNil())
				Expect(certImage).ToNot(BeNil())
				Expect(testResults).ToNot(BeNil())
			})
		})
	})
})
