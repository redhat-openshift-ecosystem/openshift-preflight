package operatorsdk

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOperatorSdk(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operator Suite")
}
