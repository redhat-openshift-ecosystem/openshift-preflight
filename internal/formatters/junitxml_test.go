package formatters

import (
	"context"
	"errors"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JUnitXML Formatter", func() {
	Context("With a valid UserResponse", func() {
		var response certification.Results
		BeforeEach(func() {
			response = certification.Results{
				TestedImage:   "example.com/repo/image:tag",
				PassedOverall: true,
				TestedOn: runtime.OpenshiftClusterVersion{
					Name:    "ClusterName",
					Version: "Clusterversion",
				},
				CertificationHash: "",
				Passed: []certification.Result{
					{
						Check: check.NewGenericCheck(
							"PassedCheck",
							func(ctx context.Context, ir image.ImageReference) (bool, error) { return true, nil },
							check.Metadata{
								Description:      "description",
								KnowledgeBaseURL: "kburl",
								CheckURL:         "checkurl",
							},
							check.HelpText{
								Message:    "helptext",
								Suggestion: "suggestion",
							},
							nil),
						ElapsedTime: 0,
					},
				},
				Failed: []certification.Result{
					{
						Check: check.NewGenericCheck(
							"FailedCheck",
							func(ctx context.Context, ir image.ImageReference) (bool, error) { return false, nil },
							check.Metadata{
								Description:      "description",
								KnowledgeBaseURL: "kburl",
								CheckURL:         "checkurl",
							},
							check.HelpText{
								Message:    "helptext",
								Suggestion: "suggestion",
							},
							nil),
						ElapsedTime: 0,
					},
				},
				Errors: []certification.Result{
					{
						Check: check.NewGenericCheck(
							"ErroredCheck",
							func(ctx context.Context, ir image.ImageReference) (bool, error) {
								return false, errors.New("someerror")
							},
							check.Metadata{
								Description:      "description",
								KnowledgeBaseURL: "kburl",
								CheckURL:         "checkurl",
							},
							check.HelpText{
								Message:    "helptext",
								Suggestion: "suggestion",
							},
							nil),
						ElapsedTime: 0,
					},
				},
				Warned: []certification.Result{
					{
						Check: check.NewGenericCheck(
							"WarningCheckPass",
							func(ctx context.Context, ir image.ImageReference) (bool, error) { return true, nil },
							check.Metadata{
								Description:      "description",
								KnowledgeBaseURL: "kburl",
								CheckURL:         "checkurl",
							},
							check.HelpText{
								Message:    "helptext",
								Suggestion: "suggestion",
							},
							nil),
						ElapsedTime: 0,
					},
					{
						Check: check.NewGenericCheck(
							"WarningCheckFail",
							func(ctx context.Context, ir image.ImageReference) (bool, error) { return false, nil },
							check.Metadata{
								Description:      "description",
								KnowledgeBaseURL: "kburl",
								CheckURL:         "checkurl",
							},
							check.HelpText{
								Message:    "helptext",
								Suggestion: "suggestion",
							},
							nil),
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
