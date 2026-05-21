package pyxis

import (
	"context"
	"net/http"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var _ = Describe("filterExcludedLayers", func() {
	var (
		excludedHash cranev1.Hash
		validHash1   cranev1.Hash
		validHash2   cranev1.Hash
	)

	BeforeEach(func() {
		var err error
		excludedHash, err = cranev1.NewHash("sha256:5f70bf18a086007016e948b04aed3b82103a36bea41755b6cddfaf10ace3c6ef")
		Expect(err).ToNot(HaveOccurred())
		validHash1, err = cranev1.NewHash("sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
		Expect(err).ToNot(HaveOccurred())
		validHash2, err = cranev1.NewHash("sha256:fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321")
		Expect(err).ToNot(HaveOccurred())
	})

	Context("when the input contains the excluded hash", func() {
		It("should filter out the excluded hash", func() {
			input := []cranev1.Hash{validHash1, excludedHash, validHash2}
			result := filterExcludedLayers(input)
			Expect(result).To(HaveLen(2), "result should contain only the two valid hashes after filtering")
			Expect(result).To(ContainElement(validHash1), "validHash1 should be present in the filtered result")
			Expect(result).To(ContainElement(validHash2), "validHash2 should be present in the filtered result")
			Expect(result).ToNot(ContainElement(excludedHash), "excludedHash should be removed from the result")
		})
	})

	Context("when the input does not contain the excluded hash", func() {
		It("should return all hashes", func() {
			input := []cranev1.Hash{validHash1, validHash2}
			result := filterExcludedLayers(input)
			Expect(result).To(HaveLen(2), "result should contain all input hashes when no excluded hash is present")
			Expect(result).To(ContainElement(validHash1), "validHash1 should be present in the result")
			Expect(result).To(ContainElement(validHash2), "validHash2 should be present in the result")
		})
	})

	Context("when the input is empty", func() {
		It("should return an empty slice", func() {
			input := []cranev1.Hash{}
			result := filterExcludedLayers(input)
			Expect(result).To(BeEmpty(), "result should be empty when input is empty")
		})
	})

	Context("when the input contains only the excluded hash", func() {
		It("should return an empty slice", func() {
			input := []cranev1.Hash{excludedHash}
			result := filterExcludedLayers(input)
			Expect(result).To(BeEmpty(), "result should be empty when input contains only the excluded hash")
		})
	})

	Context("when the input contains multiple instances of the excluded hash", func() {
		It("should filter out all instances", func() {
			input := []cranev1.Hash{validHash1, excludedHash, validHash2, excludedHash}
			result := filterExcludedLayers(input)
			Expect(result).To(HaveLen(2), "result should contain only the two valid hashes after filtering all excluded instances")
			Expect(result).To(ContainElement(validHash1), "validHash1 should be present in the filtered result")
			Expect(result).To(ContainElement(validHash2), "validHash2 should be present in the filtered result")
			Expect(result).ToNot(ContainElement(excludedHash), "all instances of excludedHash should be removed from the result")
		})
	})
})

var _ = Describe("Pyxis CheckRedHatLayers", func() {
	ctx := context.Background()
	var pyxisClient *pyxisClient
	mux := http.NewServeMux()
	mux.HandleFunc("/query/", pyxisGraphqlLayerHandler(ctx))

	Context("when some layers are provided", func() {
		BeforeEach(func() {
			pyxisClient = NewPyxisClient("my.pyxis.host/query/", "my-spiffy-api-token", "my-awesome-project-id", &http.Client{Transport: localRoundTripper{handler: mux}})
		})
		Context("and a layer is a known good layer", func() {
			It("should be a good layer", func() {
				certImages, err := pyxisClient.CertifiedImagesContainingLayers(ctx, []cranev1.Hash{{}})
				Expect(err).ToNot(HaveOccurred(), "CertifiedImagesContainingLayers should not return an error")
				Expect(certImages).ToNot(BeNil(), "certImages should not be nil")
				Expect(certImages).ToNot(BeZero(), "certImages should contain certified image data")
			})
		})
	})
})
