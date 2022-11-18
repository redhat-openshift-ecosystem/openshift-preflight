package operator

import (
	"context"
	"os"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/operatorsdk"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ScorecardBasicCheck", func() {
	var (
		scorecardBasicCheck ScorecardBasicSpecCheck
		fakeEngine          operatorSdk
		testcontext         context.Context
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
		items := []operatorsdk.OperatorSdkScorecardItem{
			{
				Status: operatorsdk.OperatorSdkScorecardStatus{
					Results: []operatorsdk.OperatorSdkScorecardResult{
						{
							Name:  "olm-bundle-validation",
							Log:   "log",
							State: "pass",
						},
					},
				},
			},
		}
		report := operatorsdk.OperatorSdkScorecardReport{
			Stdout: stdout,
			Stderr: stderr,
			Items:  items,
		}
		fakeEngine = FakeOperatorSdk{
			OperatorSdkReport: report,
		}
		scorecardBasicCheck = *NewScorecardBasicSpecCheck(fakeEngine, "myns", "mysa", []byte("fake kubeconfig contents"), "20")

		tmpDir, err := os.MkdirTemp("", "artifacts-*")
		Expect(err).ToNot(HaveOccurred())

		aw, err := artifacts.NewFilesystemWriter(artifacts.WithDirectory(tmpDir))
		Expect(err).ToNot(HaveOccurred())
		testcontext = artifacts.ContextWithWriter(context.Background(), aw)

		DeferCleanup(os.RemoveAll, tmpDir)
	})

	AssertMetaData(&scorecardBasicCheck)

	Describe("Operator Bundle Scorecard", func() {
		Context("When Operator Bundle Scorecard Basic Check has a pass", func() {
			It("Should pass Validate", func() {
				ok, err := scorecardBasicCheck.Validate(testcontext, image.ImageReference{ImageURI: "dummy/image"})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When Operator Bundle Scorecard Basic Check has a fail", func() {
			BeforeEach(func() {
				engine := fakeEngine.(FakeOperatorSdk)
				engine.OperatorSdkReport.Items[0].Status.Results[0].State = "fail"
				fakeEngine = engine
				scorecardBasicCheck = *NewScorecardBasicSpecCheck(fakeEngine, "myns", "mysa", []byte("fake kubeconfig contents"), "20")
			})
			It("Should not pass Validate", func() {
				ok, err := scorecardBasicCheck.Validate(testcontext, image.ImageReference{ImageURI: "dummy/image"})
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
	Describe("Checking that OperatorSdk errors are handled correctly", func() {
		BeforeEach(func() {
			fakeEngine = BadOperatorSdk{}
			scorecardBasicCheck = *NewScorecardBasicSpecCheck(fakeEngine, "myns", "mysa", []byte("fake kubeconfig contents"), "20")
		})
		Context("When OperatorSdk throws an error", func() {
			It("should fail Validate and return an error", func() {
				ok, err := scorecardBasicCheck.Validate(testcontext, image.ImageReference{ImageURI: "dummy/image"})
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
