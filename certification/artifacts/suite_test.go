package artifacts

import (
	"os"
	"path"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestArtifacts(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Artifacts Suite")
}

// Initialize an artifacts dir for the test suite. The base dir
// must be defined here to allow AfterSuite to remove it.
var artifactsPkgTestBaseDir string

var _ = BeforeSuite(func() {
	artifactsPkgTestBaseDir, err := os.MkdirTemp(os.TempDir(), "artifacts-pkg-test-*")
	Expect(err).ToNot(HaveOccurred())
	Expect(len(artifactsPkgTestBaseDir)).ToNot(BeZero())
	artifactsDir := path.Join(artifactsPkgTestBaseDir, "artifacts")

	// Configure artifacts directory.
	SetDir(artifactsDir)
})

var _ = AfterSuite(func() {
	err := os.RemoveAll(artifactsPkgTestBaseDir)
	Expect(err).ToNot(HaveOccurred())
})
