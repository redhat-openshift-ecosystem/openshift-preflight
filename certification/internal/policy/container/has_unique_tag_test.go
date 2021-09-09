package container

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/utils/migration"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
)

// podmanEngine is a package-level variable. In some tests, we
// override it with a "happy path" engine, that returns good data.
// In the unhappy path, we override it with an engine that returns
// nothing but errors.

var _ = Describe("UniqueTag", func() {
	var (
		hasUniqueTagCheck HasUniqueTagCheck
		fakeEngine        cli.SkopeoEngine
	)

	BeforeEach(func() {
		fakeEngine = FakeSkopeoEngine{
			SkopeoReportStdout: "",
			Tags:               validImageTags(),
			SkopeoReportStderr: "",
		}
	})
	Describe("Checking for unique tags", func() {
		Context("When it has tags other than latest", func() {
			BeforeEach(func() {
				hasUniqueTagCheck = *NewHasUniqueTagCheck(&fakeEngine)
			})
			It("should pass Validate", func() {
				ok, err := hasUniqueTagCheck.Validate(migration.ImageToImageReference("dummy/image"))
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When it has only latest tag", func() {
			BeforeEach(func() {
				engine := fakeEngine.(FakeSkopeoEngine)
				engine.SkopeoReportStdout = ""
				engine.Tags = invalidImageTags()
				engine.SkopeoReportStderr = ""
				fakeEngine = engine
				hasUniqueTagCheck = *NewHasUniqueTagCheck(&fakeEngine)
			})
			It("should not pass Validate", func() {
				log.Errorf("Run Report:")
				ok, err := hasUniqueTagCheck.Validate(migration.ImageToImageReference("dummy/image"))
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
	Describe("Checking that SkopeoEngine errors are handled correctly", func() {
		BeforeEach(func() {
			fakeEngine = BadSkopeoEngine{}
			hasUniqueTagCheck = *NewHasUniqueTagCheck(&fakeEngine)
		})
		Context("When Skopeo throws an error", func() {
			It("should fail Validate and return an error", func() {
				ok, err := hasUniqueTagCheck.Validate(migration.ImageToImageReference("dummy/image"))
				Expect(err).To(HaveOccurred())
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
