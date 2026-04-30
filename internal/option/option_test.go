package option

import (
	"context"
	"time"

	"github.com/google/go-containerregistry/pkg/crane"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakeCraneConfig struct {
	dockerConfig string
	platform     string
	insecure     bool
}

func (f *fakeCraneConfig) CraneDockerConfig() string { return f.dockerConfig }
func (f *fakeCraneConfig) CranePlatform() string     { return f.platform }
func (f *fakeCraneConfig) CraneInsecure() bool       { return f.insecure }

var _ = Describe("Option", func() {
	Describe("GenerateCraneOptions", func() {
		var cfg *fakeCraneConfig

		BeforeEach(func() {
			cfg = &fakeCraneConfig{
				dockerConfig: "",
				platform:     "amd64",
				insecure:     false,
			}
		})

		It("should return a non-empty list of crane options", func() {
			opts := GenerateCraneOptions(context.Background(), cfg)
			Expect(opts).ToNot(BeEmpty())
		})

		Context("when insecure is true", func() {
			BeforeEach(func() {
				cfg.insecure = true
			})

			It("should return additional options for insecure access", func() {
				secureOpts := GenerateCraneOptions(context.Background(), &fakeCraneConfig{
					platform: "amd64",
					insecure: false,
				})
				insecureOpts := GenerateCraneOptions(context.Background(), cfg)
				Expect(len(insecureOpts)).To(BeNumerically(">", len(secureOpts)))
			})
		})
	})

	Describe("RetryOnceAfter", func() {
		It("should return a non-nil crane.Option", func() {
			opt := RetryOnceAfter(5 * time.Second)
			Expect(opt).ToNot(BeNil())
		})

		It("should apply remote options to crane.Options when called", func() {
			opt := RetryOnceAfter(5 * time.Second)
			o := &crane.Options{}
			opt(o)
			Expect(o.Remote).ToNot(BeEmpty())
		})
	})
})
