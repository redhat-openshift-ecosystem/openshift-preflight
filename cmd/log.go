package cmd

import (
	"io"
	"os"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"

	log "github.com/sirupsen/logrus"
)

// initLogger will configure the logger so that it matches the
// user's specified configuration, such as configuring output files.
func initLogger(config runtime.Config) error {
	logname := config.CLILogFile

	logFile, err := os.OpenFile(logname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0700)
	if err == nil {
		// if we could open the file for writing, write to both stderr and the file
		// otherwise, just use stderr
		mw := io.MultiWriter(os.Stderr, logFile)
		log.SetOutput(mw)
	} else {
		log.Info("Failed to log to file, using default stderr")
	}

	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.TraceLevel)

	return nil
}
