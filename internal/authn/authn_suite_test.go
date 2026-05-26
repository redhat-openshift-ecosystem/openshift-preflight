package authn

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAuthn(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Authn Suite")
}
