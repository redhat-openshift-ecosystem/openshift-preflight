package container

import (
	"context"
	"path"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"

	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HasModifiedFiles", func() {
	var (
		hasModifiedFiles HasModifiedFilesCheck
		pkgList          packageFilesRef
	)

	BeforeEach(func() {
		pkgList = packageFilesRef{
			LayerFiles: [][]string{
				{
					"this",
					"is",
					"not",
					"prohibited",
				},
				{
					"there",
					"are",
					"no",
					"prohibited",
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

	Describe("Checking if it has any modified RPM files", func() {
		Context("When there are no modified RPM files found", func() {
			It("should pass validate", func() {
				ok, err := hasModifiedFiles.validate(&pkgList)
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
				ok, err := hasModifiedFiles.validate(&pkgs)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})

	Context("When building the installed file list for installed packages", func() {
		const (
			basename = "foobasename"
			dirname  = "foodirname"
			dirindex = 0
		)
		var goodPkgList []*rpmdb.PackageInfo
		var badPkgList []*rpmdb.PackageInfo

		BeforeEach(func() {
			goodPkgList = []*rpmdb.PackageInfo{
				{
					BaseNames:  []string{basename},
					DirIndexes: []int32{dirindex},
					DirNames:   []string{dirname},
				},
			}

			badPkgList = []*rpmdb.PackageInfo{
				{
					BaseNames:  []string{basename},
					DirIndexes: []int32{dirindex},
					DirNames:   []string{dirname, "extra"},
				},
			}
		})
		It("should contain all files installed by the package according to its metadata", func() {
			files, err := hasModifiedFiles.getInstalledFilesFor(goodPkgList)
			Expect(err).ToNot(HaveOccurred())

			_, ok := files[path.Join(dirname, basename)]
			Expect(ok).To(BeTrue())
		})

		It("should fail if the rpm is invalid", func() {
			_, err := hasModifiedFiles.getInstalledFilesFor(badPkgList)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("When evaluating files in each layer", func() {
		var files [][]string
		BeforeEach(func() {
			files = [][]string{
				{"foo"},
				{"bar"},
			}
		})
		It("should shift over to the next layer if the first layer is empty", func() {
			files[0] = []string{}
			shiftedFiles, shifted := hasModifiedFiles.dropFirstLayerIfEmpty(files)
			Expect(shiftedFiles).To(BeEquivalentTo(files[1:]))
			Expect(shifted).To(BeTrue())
		})

		It("should start at the first layer if it's not empty", func() {
			unshiftedFiles, shifted := hasModifiedFiles.dropFirstLayerIfEmpty(files)
			Expect(unshiftedFiles).To(BeEquivalentTo(files))
			Expect(shifted).To(BeFalse())
		})
	})

	Context("When calling the top level Validate", func() {
		It("should fail with an invalid ImageReference", func() {
			passed, err := hasModifiedFiles.Validate(context.TODO(), certification.ImageReference{})
			Expect(err).To(HaveOccurred())
			Expect(passed).To(BeFalse())
		})
	})

	AssertMetaData(&hasModifiedFiles)
})
