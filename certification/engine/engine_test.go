package engine

import (
	"context"

	goruntime "runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/policy"
)

var _ = Describe("CheckInitialization", func() {
	When("initializing the engine", func() {
		It("should not return an error", func() {
			_, err := New(context.TODO(), "example.com/some/image:latest", []certification.Check{}, nil, "", false, false, false, goruntime.GOARCH)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

var _ = Describe("Check Initialization", func() {
	When("initializing container checks", func() {
		It("should properly return checks for default container policy", func() {
			_, err := InitializeContainerChecks(context.TODO(), policy.PolicyContainer, ContainerCheckConfig{})
			Expect(err).ToNot(HaveOccurred())
		})
		It("should properly return checks for the scratch policy", func() {
			_, err := InitializeContainerChecks(context.TODO(), policy.PolicyScratch, ContainerCheckConfig{})
			Expect(err).ToNot(HaveOccurred())
		})
		It("should properly return checks for the root policy", func() {
			_, err := InitializeContainerChecks(context.TODO(), policy.PolicyRoot, ContainerCheckConfig{})
			Expect(err).ToNot(HaveOccurred())
		})
		It("should throw an error if the policy is unknown", func() {
			_, err := InitializeContainerChecks(context.TODO(), policy.Policy("foo"), ContainerCheckConfig{})
			Expect(err).To(HaveOccurred())
		})
	})

	When("initializing operator checks", func() {
		It("should properly return checks for the root policy", func() {
			_, err := InitializeOperatorChecks(context.TODO(), policy.PolicyOperator, OperatorCheckConfig{})
			Expect(err).ToNot(HaveOccurred())
		})
		It("should throw an error if the policy is unknown", func() {
			_, err := InitializeOperatorChecks(context.TODO(), policy.Policy("bar"), OperatorCheckConfig{})
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("Check Name Queries", func() {
	DescribeTable("The checks associated with valid policy should return the expected check names",
		func(queryFunc func(context.Context) []string, expected []string) {
			c := queryFunc(context.TODO())
			Expect(queryFunc(context.TODO())).To(ContainElements(expected))
			Expect(len(c)).To(Equal(len(expected)))
		},
		Entry("default container policy", ContainerPolicy, []string{
			"HasLicense",
			"HasUniqueTag",
			"LayerCountAcceptable",
			"HasNoProhibitedPackages",
			"HasRequiredLabel",
			"RunAsNonRoot",
			"HasModifiedFiles",
			"BasedOnUbi",
		}),
		Entry("default operator policy", OperatorPolicy, []string{
			"ScorecardBasicSpecCheck",
			"ScorecardOlmSuiteCheck",
			"DeployableByOLM",
			"ValidateOperatorBundle",
			"BundleImageRefsAreCertified",
			"SecurityContextConstraintsInCSV",
			"AllImageRefsInRelatedImages",
		}),
		Entry("scratch container policy", ScratchContainerPolicy, []string{
			"HasLicense",
			"HasUniqueTag",
			"LayerCountAcceptable",
			"HasRequiredLabel",
			"RunAsNonRoot",
		}),
		Entry("root container policy", RootExceptionContainerPolicy, []string{
			"HasLicense",
			"HasUniqueTag",
			"LayerCountAcceptable",
			"HasNoProhibitedPackages",
			"HasRequiredLabel",
			"HasModifiedFiles",
			"BasedOnUbi",
		}),
	)

	When("the policy is unknown", func() {
		It("should return an empty list", func() {
			c := checkNamesFor(context.TODO(), policy.Policy("does not exist"))
			Expect(c).To(Equal([]string{}))
		})
	})
})
