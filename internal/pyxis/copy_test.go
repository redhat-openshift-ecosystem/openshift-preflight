package pyxis

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var _ = Describe("CertificationInput.DeepCopy", func() {
	It("does not share mutable fields with the input", func() {
		var testResults TestResults
		err := json.Unmarshal([]byte(`{"results":{"passed":[{"name":"check"}]}}`), &testResults)
		Expect(err).ToNot(HaveOccurred())

		original := &CertificationInput{
			CertProject: &CertProject{
				CertificationStatus: "Started",
			},
			CertImage: &CertImage{
				Repositories: []Repository{
					{
						Registry: "registry.example.com",
						Tags: []Tag{
							{Name: "latest"},
						},
					},
				},
				ParsedData: &ParsedData{
					Labels:                 []Label{{Name: "name", Value: "original"}},
					Layers:                 []string{"original-layer"},
					UncompressedLayerSizes: []Layer{{LayerID: "original-layer", Size: 1}},
				},
				FreshnessGrades: []FreshnessGrade{{Grade: "A"}},
			},
			TestResults: &testResults,
			RpmManifest: &RPMManifest{
				RPMS: []RPM{{Name: "original-rpm"}},
			},
			Artifacts: []Artifact{{Filename: "artifact"}},
		}

		copied := original.DeepCopy()

		copied.CertProject.CertificationStatus = "In Progress"
		copied.CertImage.Repositories[0].Registry = "copy.example.com"
		copied.CertImage.Repositories[0].Tags[0].Name = "copy"
		copied.CertImage.ParsedData.Labels[0].Value = "copy"
		copied.CertImage.ParsedData.Layers[0] = "copy-layer"
		copied.CertImage.ParsedData.UncompressedLayerSizes[0].LayerID = "copy-layer"
		copied.CertImage.FreshnessGrades[0].Grade = "B"
		copied.TestResults.Results.Passed[0].Name = "copy-check"
		copied.RpmManifest.RPMS[0].Name = "copy-rpm"
		copied.Artifacts[0].ImageID = "copy-image"

		Expect(original.CertProject.CertificationStatus).To(Equal("Started"))
		Expect(original.CertImage.Repositories[0].Registry).To(Equal("registry.example.com"))
		Expect(original.CertImage.Repositories[0].Tags[0].Name).To(Equal("latest"))
		Expect(original.CertImage.ParsedData.Labels[0].Value).To(Equal("original"))
		Expect(original.CertImage.ParsedData.Layers[0]).To(Equal("original-layer"))
		Expect(original.CertImage.ParsedData.UncompressedLayerSizes[0].LayerID).To(Equal("original-layer"))
		Expect(original.CertImage.FreshnessGrades[0].Grade).To(Equal("A"))
		Expect(original.TestResults.Results.Passed[0].Name).To(Equal("check"))
		Expect(original.RpmManifest.RPMS[0].Name).To(Equal("original-rpm"))
		Expect(original.Artifacts[0].ImageID).To(BeEmpty())
	})

	It("returns nil for a nil input", func() {
		var input *CertificationInput

		Expect(input.DeepCopy()).To(BeNil())
	})

	It("handles nil nested values", func() {
		copied := (&CertificationInput{}).DeepCopy()

		Expect(copied).ToNot(BeNil())
		Expect(copied.CertProject).To(BeNil())
		Expect(copied.CertImage).To(BeNil())
		Expect(copied.TestResults).To(BeNil())
		Expect(copied.RpmManifest).To(BeNil())

		copied = (&CertificationInput{CertImage: &CertImage{}}).DeepCopy()

		Expect(copied.CertImage).ToNot(BeNil())
		Expect(copied.CertImage.ParsedData).To(BeNil())
	})
})
