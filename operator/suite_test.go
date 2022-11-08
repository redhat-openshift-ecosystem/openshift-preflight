package operator

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOperatorLib(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "lib operator suite")
}
