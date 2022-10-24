package formatters

import (
	"context"
	"errors"

	"github.com/redhat-openshift-ecosystem/preflight/certification"
	"github.com/redhat-openshift-ecosystem/preflight/certification/runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JUnitXML Formatter", func() {
	Context("With a valid UserResponse", func() {
		var response runtime.Results
		BeforeEach(func() {
			response = runtime.Results{
				TestedImage:   "example.com/repo/image:tag",
				PassedOverall: true,
				TestedOn: runtime.OpenshiftClusterVersion{
					Name:    "ClusterName",
					Version: "Clusterversion",
				},
				CertificationHash: "",
				Passed: []runtime.Result{
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
				Failed: []runtime.Result{
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
				Errors: []runtime.Result{
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
