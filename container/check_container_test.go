package container

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	preflighterr "github.com/redhat-openshift-ecosystem/openshift-preflight/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/runtime"
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
			c := NewCheck(img,
				WithCertificationProject(certproject, token),
				WithDockerConfigJSONFromFile(dockerconfigjson),
				WithPyxisHost(pyxishost),
				WithPlatform(platform),
				WithInsecureConnection(),
			)

			Expect(c.image).To(Equal(img))
			Expect(c.certificationProjectID).To(Equal(certproject))
			Expect(c.pyxisToken).To(Equal(token))
			Expect(c.dockerconfigjson).To(Equal(dockerconfigjson))
			Expect(c.pyxisHost).To(Equal(pyxishost))
			Expect(c.platform).To(Equal(platform))
			Expect(c.insecure).To(Equal(insecure))
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
})
