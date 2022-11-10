package runtime

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Runtime ReadOnlyConfig test", func() {
	Context("When calling ReadOnly on a config", func() {
		c := &Config{
			Image:                  "image",
			Policy:                 "policy",
			ResponseFormat:         "format",
			Bundle:                 true,
			Scratch:                true,
			LogFile:                "logfile",
			Artifacts:              "artifacts",
			WriteJUnit:             true,
			CertificationProjectID: "certprojid",
			PyxisHost:              "pyxishost",
			PyxisAPIToken:          "pyxisapitoken",
			DockerConfig:           "dockercfg",
			Submit:                 true,
			Platform:               "s390x",
			Insecure:               true,
			Namespace:              "ns",
			ServiceAccount:         "sa",
			ScorecardImage:         "scorecardimg",
			ScorecardWaitTime:      "waittime",
			Channel:                "channel",
			IndexImage:             "indeximg",
			Kubeconfig:             "kubeconfig",
		}
		cro := c.ReadOnly()
		It("should return values assigned to corresponding struct fields", func() {
			Expect(cro.Image()).To(Equal("image"))
			Expect(cro.Policy()).To(Equal("policy"))
			Expect(cro.ResponseFormat()).To(Equal("format"))
			Expect(cro.IsBundle()).To(Equal(true))
			Expect(cro.IsScratch()).To(Equal(true))
			Expect(cro.LogFile()).To(Equal("logfile"))
			Expect(cro.Artifacts()).To(Equal("artifacts"))
			Expect(cro.WriteJUnit()).To(Equal(true))
			Expect(cro.CertificationProjectID()).To(Equal("certprojid"))
			Expect(cro.PyxisHost()).To(Equal("pyxishost"))
			Expect(cro.PyxisAPIToken()).To(Equal("pyxisapitoken"))
			Expect(cro.DockerConfig()).To(Equal("dockercfg"))
			Expect(cro.Submit()).To(Equal(true))
			Expect(cro.Platform()).To(Equal("s390x"))
			Expect(cro.Insecure()).To(BeTrue())
			Expect(cro.Namespace()).To(Equal("ns"))
			Expect(cro.ServiceAccount()).To(Equal("sa"))
			Expect(cro.ScorecardImage()).To(Equal("scorecardimg"))
			Expect(cro.ScorecardWaitTime()).To(Equal("waittime"))
			Expect(cro.Channel()).To(Equal("channel"))
			Expect(cro.IndexImage()).To(Equal("indeximg"))
			Expect(cro.Kubeconfig()).To(Equal("kubeconfig"))
		})
	})
})
