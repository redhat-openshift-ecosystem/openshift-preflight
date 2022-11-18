package artifacts

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestArtifacts(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Artifacts Suite")
}
