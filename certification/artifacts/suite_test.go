package artifacts

import (
	"os"
	"path"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
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

	// Set the artifacts dir in viper. This won't have been created
	// prior to running tests.
	viper.Set("artifacts", artifactsDir)
})

var _ = AfterSuite(func() {
	err := os.RemoveAll(artifactsPkgTestBaseDir)
	Expect(err).ToNot(HaveOccurred())
})
