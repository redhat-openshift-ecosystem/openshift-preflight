package engine

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInternalEngine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Internal Engine Suite")
}
