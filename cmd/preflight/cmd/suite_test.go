package cmd

import (
	"archive/tar"
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCMD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CMD Suite")
}

var createAndCleanupDirForArtifactsAndLogs = func() {
	tmpDir, err := os.MkdirTemp("", "cmd-execute-*")
	Expect(err).ToNot(HaveOccurred())
	os.Setenv("PFLT_ARTIFACTS", filepath.Join(tmpDir, "artifacts"))
	os.Setenv("PFLT_LOGFILE", filepath.Join(tmpDir, "preflight.log"))
	DeferCleanup(os.RemoveAll, tmpDir)
	DeferCleanup(os.Unsetenv, "PFLT_ARTIFACTS")
	DeferCleanup(os.Unsetenv, "PFLT_LOGFILE")
}

// In order to test some negative paths, this io.Writer will just throw an error
type errWriter int

func (errWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("test error")
}

// tarEntryNames returns the base names of every entry in the tar file at
// path, for asserting on artifacts.tar contents in tests.
func tarEntryNames(path string) []string {
	f, err := os.Open(path)
	Expect(err).ToNot(HaveOccurred())
	defer f.Close()

	var names []string
	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		names = append(names, hdr.Name)
	}

	return names
}
