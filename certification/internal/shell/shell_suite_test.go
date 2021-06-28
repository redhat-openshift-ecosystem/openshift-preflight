package shell

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var (
	logger *logrus.Logger
)

func TestShell(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Shell Suite")
}

var _ = BeforeSuite(func() {
	logger = logrus.New()

	time := time.Now().Unix()
	logname := fmt.Sprintf("preflight-test-%d.log", time)
	logFile, err := os.OpenFile(logname, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0700)
	if err == nil {
		mw := io.MultiWriter(os.Stdout, logFile)
		logger.SetOutput(mw)
	} else {
		logger.Info("Failed to log to file, using default stderr")
	}
	logger.SetFormatter(&logrus.TextFormatter{})
	logger.SetLevel(logrus.TraceLevel)
})
