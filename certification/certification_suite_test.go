package certification

import (
	"testing"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

func TestCertification(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Certification Suite")
}
