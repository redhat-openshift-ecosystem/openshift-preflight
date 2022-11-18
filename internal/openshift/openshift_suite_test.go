package openshift

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOpenshift(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Openshift Client Suite")
}
