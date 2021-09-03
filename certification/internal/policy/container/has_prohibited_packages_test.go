package container

import (
	. "github.com/onsi/ginkgo"
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
	Describe("Checking if it has an prohibited packages", func() {
		Context("When there are no prohibited packages found", func() {
			It("should pass validate", func() {
				ok, err := HasNoProhibitedPackages.validate(pkgList)
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
				ok, err := HasNoProhibitedPackages.validate(pkgs)
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
				ok, err := HasNoProhibitedPackages.validate(pkgs)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
