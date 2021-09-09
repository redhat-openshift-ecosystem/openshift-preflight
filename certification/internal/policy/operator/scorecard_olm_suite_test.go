package operator

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
)

var _ = Describe("ScorecardBasicCheck", func() {
	var (
		scorecardOlmSuiteCheck ScorecardOlmSuiteCheck
		fakeEngine             cli.OperatorSdkEngine
	)

	BeforeEach(func() {
		stdout := `{
			"apiVersion": "scorecard.operatorframework.io/v1alpha3",
			"kind": "TestList",
			"items": [
			  {
				"kind": "Test",
				"apiVersion": "scorecard.operatorframework.io/v1alpha3",
				"spec": {
				  "image": "quay.io/operator-framework/scorecard-test:latest",
				  "entrypoint": [
					"scorecard-test",
					"olm-bundle-validation"
				  ],
				  "labels": {
					"suite": "olm",
					"test": "olm-bundle-validation-test"
				  }
				},
				"status": {
				  "results": [
					{
					  "name": "olm-bundle-validation",
					  "log": "time=\"2020-06-10T19:02:49Z\" level=debug msg=\"Found manifests directory\" name=bundle-test\ntime=\"2020-06-10T19:02:49Z\" level=debug msg=\"Found metadata directory\" name=bundle-test\ntime=\"2020-06-10T19:02:49Z\" level=debug msg=\"Getting mediaType info from manifests directory\" name=bundle-test\ntime=\"2020-06-10T19:02:49Z\" level=info msg=\"Found annotations file\" name=bundle-test\ntime=\"2020-06-10T19:02:49Z\" level=info msg=\"Could not find optional dependencies file\" name=bundle-test\n",
					  "state": "pass"
					}
				  ]
				}
			  }
			]
		  }
		  `
		stderr := ""
		items := []cli.OperatorSdkScorecardItem{
			{
				Status: cli.OperatorSdkScorecardStatus{
					Results: []cli.OperatorSdkScorecardResult{
						{
							Name:  "olm-bundle-validation",
							Log:   "log",
							State: "pass",
						},
					},
				},
			},
		}
		report := cli.OperatorSdkScorecardReport{
			Stdout: stdout,
			Stderr: stderr,
			Items:  items,
		}
		fakeEngine = FakeOperatorSdkEngine{
			OperatorSdkReport: report,
		}
		scorecardOlmSuiteCheck = *NewScorecardOlmSuiteCheck(&fakeEngine)
	})
	Describe("Operator Bundle Scorecard", func() {
		Context("When Operator Bundle Scorecard OLM Suite Check has a pass", func() {
			It("Should pass Validate", func() {
				ok, err := scorecardOlmSuiteCheck.Validate(certification.ImageReference{ImageURI: "dummy/image"})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When Operator Bundle Scorecard OLM Suite Check has a fail", func() {
			BeforeEach(func() {
				engine := fakeEngine.(FakeOperatorSdkEngine)
				engine.OperatorSdkReport.Items[0].Status.Results[0].State = "fail"
				fakeEngine = engine
				scorecardOlmSuiteCheck = *NewScorecardOlmSuiteCheck(&fakeEngine)
			})
			It("Should not pass Validate", func() {
				ok, err := scorecardOlmSuiteCheck.Validate(certification.ImageReference{ImageURI: "dummy/image"})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
	Describe("Checking that OperatorSdkEngine errors are handled correctly", func() {
		BeforeEach(func() {
			fakeEngine = BadOperatorSdkEngine{}
			scorecardOlmSuiteCheck = *NewScorecardOlmSuiteCheck(&fakeEngine)
		})
		Context("When OperatorSdk throws an error", func() {
			It("should fail Validate and return an error", func() {
				ok, err := scorecardOlmSuiteCheck.Validate(certification.ImageReference{ImageURI: "dummy/image"})
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
