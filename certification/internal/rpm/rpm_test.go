package rpm

import (
	"context"
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RPM", func() {
	// For base paths, we will have to navigate backwards
	// in the dir structure because the working dir in
	// tests is the base dir of the package.
	//
	// It makes sense that the fixtures moving would cause this
	// test to fail as a result.
	wd, _ := os.Getwd()
	Context("With a valid sqlite3-backed rpmdb", func() {
		basePath := path.Join(wd, "..", "..", "..", "test", "rpmdbs", "sqlite")

		It("should succeed", func() {
			packages, err := GetPackageList(context.TODO(), basePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(packages)).To(Equal(1))
		})
	})

	Context("With a valid berkeleydb rpmdb", func() {
		basePath := path.Join(wd, "..", "..", "..", "test", "rpmdbs", "bdb")

		It("should succeed", func() {
			packages, err := GetPackageList(context.TODO(), basePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(packages)).To(Equal(1))
		})
	})

	Context("With no rpmdb of any kind found at known good locations", func() {
		basePath := path.Join(wd, "..", "..", "..", "test", "rpmdbs") // there's nothing here!

		It("should throw an error", func() {
			packages, err := GetPackageList(context.TODO(), basePath)
			Expect(err).To(HaveOccurred())
			Expect(len(packages)).To(Equal(0)) // basically a nil check
		})
	})

	Context("With a corrupt bdb rpmdb preventing open", func() {
		// Packages was replaced with jsut the string "CORRUPT"
		basePath := path.Join(wd, "..", "..", "..", "test", "rpmdbs", "bdb-corrupt")
		It("should throw an error", func() {
			packages, err := GetPackageList(context.TODO(), basePath)
			Expect(err).To(HaveOccurred())
			Expect(len(packages)).To(Equal(0)) // basically a nil check
		})
	})

	Context("With a corrupt sqlite rpmdb preventing open", func() {
		// rpmdb.sqlite was replaced with just the string "CORRUPT"
		basePath := path.Join(wd, "..", "..", "..", "test", "rpmdbs", "sqlite-corrupt")
		It("should throw an error", func() {
			packages, err := GetPackageList(context.TODO(), basePath)
			Expect(err).To(HaveOccurred())
			Expect(len(packages)).To(Equal(0)) // basically a nil check
		})
	})

	/* Commented until we can identify how to properly fail a read in bdb
	Context("With a corrupt bdb rpmdb preventing ListPackages", func() {
		// Packages file replaces with Requirename file... but this doesn't work.
		basePath := path.Join(wd, "..", "..", "..", "test", "rpmdbs", "bdb-fail-list-pkg")
		It("should throw an error", func() {
			packages, err := GetPackageList(context.TODO(), basePath)
			Expect(err).To(HaveOccurred())
			Expect(len(packages)).To(Equal(0)) // basically a nil check
		})
	})
	*/

	/* Commented pending bugfix on go-rpmdb
	Context("With a corrupt sqlite rpmdb preventing ListPackages", func() {
		// rpmdb.sqlite created with arbitrary Packages table that doesn't match schema.
		basePath := path.Join(wd, "..", "..", "..", "test", "rpmdbs", "sqlite-fail-list-pkg")
		It("should throw an error", func() {
			packages, err := GetPackageList(context.TODO(), basePath)
			Expect(err).To(HaveOccurred())
			Expect(len(packages)).To(Equal(0)) // basically a nil check
		})
	})
	*/
})
