package container

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	preflighterr "github.com/redhat-openshift-ecosystem/openshift-preflight/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/lib"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/runtime"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/test"
)

var _ = Describe("Container Check initialization", func() {
	When("Using options to initialize a check", func() {
		It("Should properly store the options with their correct values", func() {
			img := "placeholder"
			certproject := "certproject"
			token := "token"
			dockerconfigjson := "dockerconfig.json"
			pyxishost := "pyxishost"
			platform := "arm64"
			insecure := true
			manfiestListDigest := "12345"
			konflux := true
			c := NewCheck(img,
				WithCertificationProject(certproject, token),
				WithDockerConfigJSONFromFile(dockerconfigjson),
				WithPyxisHost(pyxishost),
				WithPlatform(platform),
				WithInsecureConnection(),
				WithManifestListDigest(manfiestListDigest),
				WithKonflux(),
			)

			Expect(c.image).To(Equal(img))
			Expect(c.certificationProjectID).To(Equal(certproject))
			Expect(c.pyxisToken).To(Equal(token))
			Expect(c.dockerconfigjson).To(Equal(dockerconfigjson))
			Expect(c.pyxisHost).To(Equal(pyxishost))
			Expect(c.platform).To(Equal(platform))
			Expect(c.insecure).To(Equal(insecure))
			Expect(c.konflux).To(Equal(konflux))
		})
		Context("with the WithCertificationComponent option", func() {
			It("should set the project ID and token", func() {
				c := NewCheck("placeholder",
					WithCertificationComponent("mycomponent", "mytoken"),
				)
				Expect(c.certificationProjectID).To(Equal("mycomponent"))
				Expect(c.pyxisToken).To(Equal("mytoken"))
			})
		})
		Context("with the pyxisenv option", func() {
			var env string
			It("should resolve the env if valid", func() {
				env = "dev"
				ph := runtime.PyxisHostLookup(env, "")
				c := NewCheck("placeholder", WithPyxisEnv(env))
				Expect(c.pyxisHost).To(Equal(ph))
			})

			It("should return the prod host if the env is invalid", func() {
				env = "invalid"
				ph := runtime.PyxisHostLookup(env, "")
				c := NewCheck("placeholder", WithPyxisEnv(env))
				Expect(c.pyxisHost).To(Equal(ph))
			})
		})
	})
})

var _ = Describe("Container Check Execution", func() {
	When("testing against a known-good image", func() {
		var chk *containerCheck
		goodImage := "quay.io/opdev/simple-demo-operator:latest"
		BeforeEach(func() {
			chk = NewCheck(goodImage)
		})

		It("Should resolve checks without issue", func() {
			ctx := context.TODO()
			err := chk.resolve(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(chk.policy).To(Equal("container"))
			Expect(chk.resolved).To(Equal(true))
			Expect(len(chk.checks)).To(Equal(10))
		})

		It("Should list checks without issue", func() {
			ctx := context.TODO()
			policy, checks, err := chk.List(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(policy).To(Equal("container"))
			Expect(len(checks)).To(Equal(10))
		})

		It("Should run without issue", func() {
			ctx := test.NewTestLoggerContext(context.TODO())
			results, err := chk.Run(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).ToNot(Equal(certification.Results{}))
			Expect(results.TestedImage).To(Equal(goodImage))
			Expect(len(results.Failed)).To(Equal(0))
			Expect(len(results.Errors)).To(Equal(0))
		})
	})

	When("testing against a known good image and konflux is true", func() {
		var chk *containerCheck
		goodImage := "quay.io/opdev/simple-demo-operator:latest"
		BeforeEach(func() {
			chk = NewCheck(goodImage)
			chk.konflux = true
		})

		It("Should resolve checks without issue", func() {
			ctx := context.TODO()
			err := chk.resolve(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(chk.policy).To(Equal("konflux"))
			Expect(chk.resolved).To(Equal(true))
			Expect(len(chk.checks)).To(Equal(8))
		})

		It("Should list checks without issue", func() {
			ctx := context.TODO()
			policy, checks, err := chk.List(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(policy).To(Equal("konflux"))
			Expect(len(checks)).To(Equal(8))
		})

		It("Should run without issue", func() {
			ctx := context.TODO()
			results, err := chk.Run(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).ToNot(Equal(certification.Results{}))
			Expect(results.TestedImage).To(Equal(goodImage))
			Expect(len(results.Failed)).To(Equal(0))
			Expect(len(results.Errors)).To(Equal(0))
		})
	})

	When("Calling the check", func() {
		It("should fail if you passed an empty image", func() {
			chk := NewCheck("")
			_, err := chk.Run(context.TODO())
			Expect(err).To(MatchError(preflighterr.ErrImageEmpty))
		})

		It("should fail if it cannot use your provided pyxis data to resolve the policy", func() {
			// This test isn't ideal because it's slow due to actually trying to use the creds to talk to Pyxis.
			chk := NewCheck("placeholder", WithPyxisEnv("dev"), WithCertificationProject("00000", "11111"))
			_, err := chk.Run(context.TODO())
			Expect(err).To(MatchError(preflighterr.ErrCannotResolvePolicyException))
		})
	})

	When("checking for bundle projects", func() {
		It("should fail if the project is an operator bundle image", func() {
			fakeClient := &fakePyxisClient{
				getProjectFunc: returnBundleProject,
			}
			chk := NewCheck("placeholder", withPyxisClient(fakeClient))
			_, err := chk.Run(context.TODO())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("bundle project detected"))
			Expect(err.Error()).To(ContainSubstring("operator certification workflow"))
		})

		It("should succeed if the project is a container project", func() {
			fakeClient := &fakePyxisClient{
				getProjectFunc: returnContainerProject,
			}
			goodImage := "quay.io/opdev/simple-demo-operator:latest"
			chk := NewCheck(goodImage, withPyxisClient(fakeClient))
			results, err := chk.Run(context.TODO())
			Expect(err).ToNot(HaveOccurred())
			Expect(results).ToNot(Equal(certification.Results{}))
			Expect(results.TestedImage).To(Equal(goodImage))
		})
	})
})

// withPyxisClient injects a pyxis client for testing purposes.
// This is not exported and should only be used in tests.
func withPyxisClient(client lib.PyxisClient) Option {
	return func(cc *containerCheck) {
		cc.pyxisClient = client
	}
}
