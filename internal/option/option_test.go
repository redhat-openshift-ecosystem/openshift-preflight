package option

import (
	"time"

	"github.com/google/go-containerregistry/pkg/crane"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Option", func() {
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
