package container

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HasNoProhibitedPackages", func() {
	var (
		HasNoProhibitedPackages HasNoProhibitedPackagesCheck
		pkgList                 []string
	)

	BeforeEach(func() {
		pkgList = []string{
			"this",
			"is",
			"not",
			"prohibitted",
		}
	})

	Context("When checking metadata", func() {
		Context("The check name should not be empty", func() {
			Expect(HasNoProhibitedPackages.Name()).ToNot(BeEmpty())
		})

		Context("The metadata keys should not be empty", func() {
			meta := HasNoProhibitedPackages.Metadata()
			Expect(meta.CheckURL).ToNot(BeEmpty())
			Expect(meta.Description).ToNot(BeEmpty())
			Expect(meta.KnowledgeBaseURL).ToNot(BeEmpty())
			// Level is optional.
		})

		Context("The help text should not be empty", func() {
			help := HasNoProhibitedPackages.Help()
			Expect(help.Message).ToNot(BeEmpty())
			Expect(help.Suggestion).ToNot(BeEmpty())
		})
	})

	Describe("Checking if it has an prohibited packages", func() {
		Context("When there are no prohibited packages found", func() {
			It("should pass validate", func() {
				ok, err := HasNoProhibitedPackages.validate(context.TODO(), pkgList)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When there was a prohibited packages found", func() {
			var pkgs []string
			BeforeEach(func() {
				pkgs = append(pkgList, "grub")
			})
			It("should not pass Validate", func() {
				ok, err := HasNoProhibitedPackages.validate(context.TODO(), pkgs)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		Context("When there is a prohibited package in the glob list found", func() {
			var pkgs []string
			BeforeEach(func() {
				pkgs = append(pkgList, "kpatch2121")
			})
			It("should not pass Validate", func() {
				ok, err := HasNoProhibitedPackages.validate(context.TODO(), pkgs)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
