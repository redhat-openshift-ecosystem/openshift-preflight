package formatters

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFormatters(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Formatters Suite")
}
