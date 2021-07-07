package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

func init() {
	time := time.Now().Unix()
	logname := fmt.Sprintf("preflight-%d.log", time)
	logFile, err := os.OpenFile(logname, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0700)
	if err == nil {
		mw := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(mw)
	} else {
		log.Info("Failed to log to file, using default stderr")
	}
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.TraceLevel)
}
