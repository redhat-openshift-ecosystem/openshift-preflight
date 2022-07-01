package container

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HasModifiedFiles", func() {
	var (
		HasModifiedFiles HasModifiedFilesCheck
		pkgList          packageFilesRef
	)

	BeforeEach(func() {
		pkgList = packageFilesRef{
			LayerFiles: [][]string{
				{
					"this",
					"is",
					"not",
					"prohibitted",
				},
				{
					"there",
					"are",
					"no",
					"prohibitted",
					"duplicates",
				},
			},
			PackageFiles: map[string]struct{}{
				"this": {},
				"is":   {},
				"not":  {},
			},
		}
	})

	Context("When checking metadata", func() {
		Context("The check name should not be empty", func() {
			Expect(HasModifiedFiles.Name()).ToNot(BeEmpty())
		})

		Context("The metadata keys should not be empty", func() {
			meta := HasModifiedFiles.Metadata()
			Expect(meta.CheckURL).ToNot(BeEmpty())
			Expect(meta.Description).ToNot(BeEmpty())
			Expect(meta.KnowledgeBaseURL).ToNot(BeEmpty())
			// Level is optional.
		})

		Context("The help text should not be empty", func() {
			help := HasModifiedFiles.Help()
			Expect(help.Message).ToNot(BeEmpty())
			Expect(help.Suggestion).ToNot(BeEmpty())
		})
	})

	Describe("Checking if it has any modified RPM files", func() {
		Context("When there are no modified RPM files found", func() {
			It("should pass validate", func() {
				ok, err := HasModifiedFiles.validate(&pkgList)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When there is a modified RPM file found", func() {
			var pkgs packageFilesRef
			BeforeEach(func() {
				pkgs = pkgList
				pkgs.LayerFiles[1] = append(pkgs.LayerFiles[1], "this")
			})
			It("should not pass Validate", func() {
				ok, err := HasModifiedFiles.validate(&pkgs)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
