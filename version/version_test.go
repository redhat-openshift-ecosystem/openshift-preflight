package version

import (
	"reflect"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("version package utility", func() {
	// Values assumed to be passed when calling make test.
	ldflagVersionOverride := "foo"
	ldflagCommitOverride := "bar"

	// These tests validate that we can override the version and commit information successfully,
	// and that our string representation includes that information.
	Context("When being passed version and commit information via ldflags", func() {
		It("should contain the passed in version and commit information in internal data structures", func() {
			Expect(Version.Version).To(Equal(ldflagVersionOverride))
			Expect(Version.Commit).To(Equal(ldflagCommitOverride))
		})
	})

	Context("When printing the VersionContext", func() {
		It("should display the version and the commit information as a string", func() {
			Expect(strings.Contains(Version.String(), ldflagVersionOverride)).To(BeTrue())
			Expect(strings.Contains(Version.String(), ldflagCommitOverride)).To(BeTrue())
		})
	})

	// These tests confirm that we have appropriate JSON struct tags because we include
	// this in Preflight Results.
	Context("When using a VersionContext", func() {
		It("should have JSON struct tags on fields", func() {
			nf, nexists := reflect.TypeOf(&Version).Elem().FieldByName("Name") // The struct key!
			Expect(nexists).To(BeTrue())
			Expect(string(nf.Tag)).To(Equal(`json:"name"`)) // the tag

			vf, vexists := reflect.TypeOf(&Version).Elem().FieldByName("Version")
			Expect(vexists).To(BeTrue())
			Expect(string(vf.Tag)).To(Equal(`json:"version"`))

			cf, cexists := reflect.TypeOf(&Version).Elem().FieldByName("Commit")
			Expect(cexists).To(BeTrue())
			Expect(string(cf.Tag)).To(Equal(`json:"commit"`))
		})

		It("should only have three struct keys for tests to be valid", func() {
			keys := reflect.TypeOf(Version).NumField()
			Expect(keys).To(Equal(3))
		})
	})
})
