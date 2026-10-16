package container

import (
	"context"
	"errors"
	"time"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	fakecranev1 "github.com/google/go-containerregistry/pkg/v1/fake"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/pyxis"
)

type fakeCertifiedRepositoriesFinder struct {
	repositories []pyxis.CertRepository
	err          error
}

func (f *fakeCertifiedRepositoriesFinder) CertifiedRepositoriesForLayers(context.Context, []cranev1.Hash) ([]pyxis.CertRepository, error) {
	return f.repositories, f.err
}

var _ = Describe("HasSupportedRedHatBaseImage", func() {
	var (
		hasSupportedBaseImage HasSupportedRedHatBaseImageCheck
		imageRef              image.ImageReference
		finder                *fakeCertifiedRepositoriesFinder
	)

	now := time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC)

	BeforeEach(func() {
		finder = &fakeCertifiedRepositoriesFinder{}
		hasSupportedBaseImage = *NewHasSupportedRedHatBaseImageCheck(finder, func() time.Time { return now })

		fakeImage := fakecranev1.FakeImage{
			ConfigFileStub: func() (*cranev1.ConfigFile, error) {
				return &cranev1.ConfigFile{
					RootFS: cranev1.RootFS{DiffIDs: []cranev1.Hash{{}}},
				}, nil
			},
		}
		imageRef.ImageInfo = &fakeImage
	})

	Describe("Checking the matched Red Hat repository", func() {
		It("passes when a repository is GA and not EOL", func() {
			finder.repositories = []pyxis.CertRepository{{
				ReleaseCategories: []string{"Generally Available"},
			}}

			ok, err := hasSupportedBaseImage.Validate(context.TODO(), imageRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		DescribeTable("does not pass without a supported repository", func(repository pyxis.CertRepository) {
			finder.repositories = []pyxis.CertRepository{repository}

			ok, err := hasSupportedBaseImage.Validate(context.TODO(), imageRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeFalse())
		},
			Entry("when the repository is EOL", pyxis.CertRepository{
				EOLDate:           &now,
				ReleaseCategories: []string{"Generally Available"},
			}),
			Entry("when the repository is not GA", pyxis.CertRepository{
				ReleaseCategories: []string{"Beta"},
			}),
		)

		It("passes when any matched repository is supported", func() {
			eolDate := now
			finder.repositories = []pyxis.CertRepository{
				{EOLDate: &eolDate, ReleaseCategories: []string{"Generally Available"}},
				{ReleaseCategories: []string{"Generally Available"}},
			}

			ok, err := hasSupportedBaseImage.Validate(context.TODO(), imageRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("does not pass when no repository matches", func() {
			ok, err := hasSupportedBaseImage.Validate(context.TODO(), imageRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})

	Context("when the Pyxis query fails", func() {
		BeforeEach(func() {
			finder.err = errors.New("pyxis is unavailable")
		})

		It("returns an error", func() {
			ok, err := hasSupportedBaseImage.Validate(context.TODO(), imageRef)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("pyxis is unavailable"))
			Expect(ok).To(BeFalse())
		})
	})

	Context("when the image config cannot be read", func() {
		BeforeEach(func() {
			imageRef.ImageInfo = &fakecranev1.FakeImage{
				ConfigFileStub: func() (*cranev1.ConfigFile, error) {
					return nil, errors.New("config error")
				},
			}
		})

		It("returns an error", func() {
			ok, err := hasSupportedBaseImage.Validate(context.TODO(), imageRef)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not get image layers"))
			Expect(ok).To(BeFalse())
		})
	})

	Context("when image information is missing", func() {
		BeforeEach(func() {
			imageRef.ImageInfo = nil
		})

		It("returns an error", func() {
			ok, err := hasSupportedBaseImage.Validate(context.TODO(), imageRef)
			Expect(err).To(MatchError("could not get image layers: image information is required"))
			Expect(ok).To(BeFalse())
		})
	})

	AssertMetaData(&hasSupportedBaseImage)

	It("uses the current time when no clock is provided", func() {
		check := NewHasSupportedRedHatBaseImageCheck(finder, nil)

		Expect(check).ToNot(BeNil())
		Expect(check.now).ToNot(BeNil())
	})

	It("returns nil for RequiredFilePatterns", func() {
		Expect(hasSupportedBaseImage.RequiredFilePatterns()).To(BeNil())
	})
})
