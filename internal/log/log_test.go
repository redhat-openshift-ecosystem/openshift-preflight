package log

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Internal Logrus Instance", func() {
	When("Changing the configuration of the instance", func() {
		L().SetFormatter(&logrus.JSONFormatter{})
		It("Should persist when called again", func() {
			Expect(l.Formatter).To(BeEquivalentTo(&logrus.JSONFormatter{}))
		})
	})
})
