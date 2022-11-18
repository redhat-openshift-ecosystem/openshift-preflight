package formatters

import (
	"context"
	"errors"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JUnitXML Formatter", func() {
	Context("With a valid UserResponse", func() {
		var response certification.Results
		BeforeEach(func() {
			response = certification.Results{
				TestedImage:       "example.com/repo/image:tag",
				PassedOverall:     true,
				TestedOn:          "ClusterName/Clusterversion",
				CertificationHash: "",
				Passed: []certification.Result{
					{
						Check: certification.NewGenericCheck(
							"PassedCheck",
							func(ctx context.Context, ir certification.ImageReference) (bool, error) { return true, nil },
							certification.Metadata{
								Description:      "description",
								KnowledgeBaseURL: "kburl",
								CheckURL:         "checkurl",
							},
							certification.HelpText{
								Message:    "helptext",
								Suggestion: "suggestion",
							}),
						ElapsedTime: 0,
					},
				},
				Failed: []certification.Result{
					{
						Check: certification.NewGenericCheck(
							"FailedCheck",
							func(ctx context.Context, ir certification.ImageReference) (bool, error) { return false, nil },
							certification.Metadata{
								Description:      "description",
								KnowledgeBaseURL: "kburl",
								CheckURL:         "checkurl",
							},
							certification.HelpText{
								Message:    "helptext",
								Suggestion: "suggestion",
							}),
						ElapsedTime: 0,
					},
				},
				Errors: []certification.Result{
					{
						Check: certification.NewGenericCheck(
							"ErroredCheck",
							func(ctx context.Context, ir certification.ImageReference) (bool, error) {
								return false, errors.New("someerror")
							},
							certification.Metadata{
								Description:      "description",
								KnowledgeBaseURL: "kburl",
								CheckURL:         "checkurl",
							},
							certification.HelpText{
								Message:    "helptext",
								Suggestion: "suggestion",
							}),
						ElapsedTime: 0,
					},
				},
			}
		})
		It("should format without error", func() {
			out, err := junitXMLFormatter(context.TODO(), response)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(out)).To(ContainSubstring("PassedCheck"))
			Expect(string(out)).To(ContainSubstring("FailedCheck"))
			Expect(string(out)).To(ContainSubstring("ErroredCheck"))
		})
	})
})
