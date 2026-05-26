package formatters

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
)

var _ = Describe("Generic formatters", func() {
	generateTestResults := func(image string, passed bool) certification.Results {
		return certification.Results{
			TestedImage:   image,
			PassedOverall: passed,
			Passed: []certification.Result{
				{
					Check:       check.NewGenericCheck("passed1", nil, check.Metadata{}, check.HelpText{}, nil),
					ElapsedTime: 1000 * time.Millisecond,
				},
			},
			Failed: []certification.Result{
				{
					Check:       check.NewGenericCheck("failed1", nil, check.Metadata{}, check.HelpText{}, nil),
					ElapsedTime: 1001 * time.Millisecond,
				},
			},
		}
	}

	Describe("genericJSONFormatter", func() {
		AfterEach(func() {
			jsonMarshalIndent = json.MarshalIndent
		})

		DescribeTable("formatting results as JSON",
			func(image string, passed bool, marshalFailure bool, expectedErrSubstring string) {
				results := generateTestResults(image, passed)

				if marshalFailure {
					jsonMarshalIndent = func(v any, prefix, indent string) ([]byte, error) {
						return nil, errors.New("this is an error")
					}
				}

				funcOutput, err := genericJSONFormatter(context.TODO(), results)

				if marshalFailure {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(expectedErrSubstring))
					return
				}

				Expect(err).ToNot(HaveOccurred())

				var testResponseObj UserResponse
				Expect(json.Unmarshal(funcOutput, &testResponseObj)).To(Succeed())

				Expect(testResponseObj.Image).To(Equal(results.TestedImage))
				Expect(testResponseObj.Passed).To(Equal(results.PassedOverall))

				for index, i := range results.Passed {
					Expect(testResponseObj.Results.Passed[index].Name).To(Equal(i.Name()))
					Expect(testResponseObj.Results.Passed[index].ElapsedTime).To(Equal(float64(i.ElapsedTime / time.Millisecond)))
				}
				for index, i := range results.Failed {
					Expect(testResponseObj.Results.Failed[index].Name).To(Equal(i.Name()))
					Expect(testResponseObj.Results.Failed[index].ElapsedTime).To(Equal(float64(i.ElapsedTime / time.Millisecond)))
				}
			},
			Entry("with passing results", "image1", true, false, ""),
			Entry("with failing results", "image2", false, false, ""),
			Entry("when MarshalIndent fails", "image3", true, true, "this is an error"),
		)
	})

	Describe("genericXMLFormatter", func() {
		AfterEach(func() {
			xmlMarshalIndent = xml.MarshalIndent
		})

		DescribeTable("formatting results as XML",
			func(image string, passed bool, marshalFailure bool, expectedErrSubstring string) {
				results := generateTestResults(image, passed)

				if marshalFailure {
					xmlMarshalIndent = func(v any, prefix, indent string) ([]byte, error) {
						return nil, errors.New("this is an error")
					}
				}

				funcOutput, err := genericXMLFormatter(context.TODO(), results)

				if marshalFailure {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(expectedErrSubstring))
					return
				}

				Expect(err).ToNot(HaveOccurred())

				var testResponseObj UserResponse
				Expect(xml.Unmarshal(funcOutput, &testResponseObj)).To(Succeed())

				Expect(testResponseObj.Image).To(Equal(results.TestedImage))
				Expect(testResponseObj.Passed).To(Equal(results.PassedOverall))

				for index, i := range results.Passed {
					Expect(testResponseObj.Results.Passed[index].Name).To(Equal(i.Name()))
					Expect(testResponseObj.Results.Passed[index].ElapsedTime).To(Equal(float64(i.ElapsedTime / time.Millisecond)))
				}
				for index, i := range results.Failed {
					Expect(testResponseObj.Results.Failed[index].Name).To(Equal(i.Name()))
					Expect(testResponseObj.Results.Failed[index].ElapsedTime).To(Equal(float64(i.ElapsedTime / time.Millisecond)))
				}
			},
			Entry("with passing results", "image1", true, false, ""),
			Entry("with failing results", "image2", false, false, ""),
			Entry("when MarshalIndent fails", "image3", true, true, "this is an error"),
		)
	})
})
