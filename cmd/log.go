package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	logger = logrus.New()

	time := time.Now().Unix()
	logname := fmt.Sprintf("preflight-%d.log", time)
	logFile, err := os.OpenFile(logname, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0700)
	if err == nil {
		mw := io.MultiWriter(os.Stdout, logFile)
		logger.SetOutput(mw)
	} else {
		logger.Info("Failed to log to file, using default stderr")
	}
	logger.SetFormatter(&logrus.TextFormatter{})
	logger.SetLevel(logrus.TraceLevel)
}
