package container

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
)

// podmanEngine is a package-level variable. In some tests, we
// override it with a "happy path" engine, that returns good data.
// In the unhappy path, we override it with an engine that returns
// nothing but errors.

type fakeTagLister struct {
	Tags []string
}

func (ftl *fakeTagLister) ListTags(imageUri string) ([]string, error) {
	return ftl.Tags, nil
}

var _ = Describe("UniqueTag", func() {
	var hasUniqueTagCheck HasUniqueTagCheck

	Describe("Checking for unique tags", func() {
		Context("When it has tags other than latest", func() {
			BeforeEach(func() {
				hasUniqueTagCheck = *NewHasUniqueTagCheck(&fakeTagLister{Tags: validImageTags()})
			})
			It("should pass Validate", func() {
				ok, err := hasUniqueTagCheck.Validate(certification.ImageReference{ImageRegistry: "index.docker.io", ImageRepository: "dummy/image"})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When it has only latest tag", func() {
			BeforeEach(func() {
				hasUniqueTagCheck = *NewHasUniqueTagCheck(&fakeTagLister{Tags: invalidImageTags()})
			})
			It("should not pass Validate", func() {
				ok, err := hasUniqueTagCheck.Validate(certification.ImageReference{ImageRegistry: "index.docker.io", ImageRepository: "dummy/other-image"})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})

func validImageTags() []string {
	return []string{"0.0.1", "0.0.2", "latest"}
}

func invalidImageTags() []string {
	return []string{"latest"}
}
