package container

import (
	"context"
	"errors"

	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
)

var _ = Describe("HasNoProhibitedPackages", func() {
	var (
		hasNoProhibitedPackages HasNoProhibitedPackagesCheck
		pkgList                 []string
	)

	BeforeEach(func() {
		pkgList = []string{
			"this",
			"is",
			"not",
			"prohibited",
		}
	})

	AssertMetaData(&hasNoProhibitedPackages)

	Describe("Checking if it has an prohibited packages", func() {
		Context("When there are no prohibited packages found", func() {
			It("should pass validate", func() {
				ok, err := hasNoProhibitedPackages.validate(context.TODO(), pkgList)
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
				ok, err := hasNoProhibitedPackages.validate(context.TODO(), pkgs)
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
				ok, err := hasNoProhibitedPackages.validate(context.TODO(), pkgs)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})

	Describe("Validate with stubbed package list", func() {
		Context("When GetPackageList returns no prohibited packages", func() {
			BeforeEach(func() {
				hasNoProhibitedPackages = HasNoProhibitedPackagesCheck{
					getPackageList: func(_ context.Context, _ string) ([]*rpmdb.PackageInfo, error) {
						return []*rpmdb.PackageInfo{
							{Name: "bash"},
							{Name: "coreutils"},
							{Name: "glibc"},
						}, nil
					},
				}
			})

			It("should pass Validate", func() {
				ok, err := hasNoProhibitedPackages.Validate(context.TODO(), image.ImageReference{ImageFSPath: "/fake"})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})

		Context("When GetPackageList returns a prohibited package", func() {
			BeforeEach(func() {
				hasNoProhibitedPackages = HasNoProhibitedPackagesCheck{
					getPackageList: func(_ context.Context, _ string) ([]*rpmdb.PackageInfo, error) {
						return []*rpmdb.PackageInfo{
							{Name: "bash"},
							{Name: "kernel"},
						}, nil
					},
				}
			})

			It("should not pass Validate", func() {
				ok, err := hasNoProhibitedPackages.Validate(context.TODO(), image.ImageReference{ImageFSPath: "/fake"})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})

		Context("When GetPackageList returns a glob-prohibited package", func() {
			BeforeEach(func() {
				hasNoProhibitedPackages = HasNoProhibitedPackagesCheck{
					getPackageList: func(_ context.Context, _ string) ([]*rpmdb.PackageInfo, error) {
						return []*rpmdb.PackageInfo{
							{Name: "bash"},
							{Name: "kpatch-dnf"},
						}, nil
					},
				}
			})

			It("should not pass Validate", func() {
				ok, err := hasNoProhibitedPackages.Validate(context.TODO(), image.ImageReference{ImageFSPath: "/fake"})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})

		Context("When GetPackageList returns an error", func() {
			BeforeEach(func() {
				hasNoProhibitedPackages = HasNoProhibitedPackagesCheck{
					getPackageList: func(_ context.Context, _ string) ([]*rpmdb.PackageInfo, error) {
						return nil, errors.New("rpm db not found")
					},
				}
			})

			It("should return an error", func() {
				ok, err := hasNoProhibitedPackages.Validate(context.TODO(), image.ImageReference{ImageFSPath: "/fake"})
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})

		Context("When GetPackageList returns an empty list", func() {
			BeforeEach(func() {
				hasNoProhibitedPackages = HasNoProhibitedPackagesCheck{
					getPackageList: func(_ context.Context, _ string) ([]*rpmdb.PackageInfo, error) {
						return []*rpmdb.PackageInfo{}, nil
					},
				}
			})

			It("should pass Validate", func() {
				ok, err := hasNoProhibitedPackages.Validate(context.TODO(), image.ImageReference{ImageFSPath: "/fake"})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
	})
})
