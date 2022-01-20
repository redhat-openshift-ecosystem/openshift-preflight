package cmd

import (
	"io"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func resultsFilenameWithExtension(ext string) string {
	return strings.Join([]string{"results", ext}, ".")
}

// preRunConfig is used by cobra.PreRun in all non-root commands to load all necessary configurations
func preRunConfig(cmd *cobra.Command, args []string) {
	// set up ENV var support
	viper.SetEnvPrefix("pflt")
	viper.AutomaticEnv()

	// set up optional config file support
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	configFileUsed := true
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			configFileUsed = false
		}
	}

	// Set up logging config defaults
	viper.SetDefault("logfile", DefaultLogFile)
	viper.SetDefault("loglevel", DefaultLogLevel)
	viper.SetDefault("artifacts", DefaultArtifactsDir)

	// Set up cluster defaults
	viper.SetDefault("namespace", DefaultNamespace)
	viper.SetDefault("serviceaccount", DefaultServiceAccount)

	// Set up scorecard wait time default
	viper.SetDefault("scorecard_wait_time", DefaultScorecardWaitTime)

	// set up logging
	logname := viper.GetString("logfile")
	logFile, err := os.OpenFile(logname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err == nil {
		mw := io.MultiWriter(os.Stderr, logFile)
		log.SetOutput(mw)
	} else {
		log.Info("Failed to log to file, using default stderr")
	}
	if ll, err := log.ParseLevel(viper.GetString("loglevel")); err == nil {
		log.SetLevel(ll)
	}

	log.SetFormatter(&log.TextFormatter{})
	if !configFileUsed {
		log.Info("config file not found, proceeding without it")
	}
}
