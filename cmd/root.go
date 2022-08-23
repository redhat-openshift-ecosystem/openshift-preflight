// Package cmd implements the command-line interface for Preflight.
package cmd

import (
	"context"
	"io"
	"os"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configFileUsed bool

func init() {
	cobra.OnInitialize(initConfig)
}

func rootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:              "preflight",
		Short:            "Preflight Red Hat certification prep tool.",
		Long:             "A utility that allows you to pre-test your bundles, operators, and container before submitting for Red Hat Certification.",
		Version:          version.Version.String(),
		Args:             cobra.MinimumNArgs(1),
		PersistentPreRun: preRunConfig,
	}

	rootCmd.PersistentFlags().String("logfile", "", "Where the execution logfile will be written. (env: PFLT_LOGFILE)")
	_ = viper.BindPFlag("logfile", rootCmd.PersistentFlags().Lookup("logfile"))

	rootCmd.PersistentFlags().String("loglevel", "", "The verbosity of the preflight tool itself. Ex. warn, debug, trace, info, error. (env: PFLT_LOGLEVEL)")
	_ = viper.BindPFlag("loglevel", rootCmd.PersistentFlags().Lookup("loglevel"))

	rootCmd.AddCommand(checkCmd())
	rootCmd.AddCommand(listChecksCmd())
	rootCmd.AddCommand(runtimeAssetsCmd())
	rootCmd.AddCommand(supportCmd())

	return rootCmd
}

func Execute() error {
	return rootCmd().ExecuteContext(context.Background())
}

func initConfig() {
	// set up ENV var support
	viper.SetEnvPrefix("pflt")
	viper.AutomaticEnv()

	// set up optional config file support
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	configFileUsed = true
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			configFileUsed = false
		}
	}

	// Set up logging config defaults
	viper.SetDefault("logfile", DefaultLogFile)
	viper.SetDefault("loglevel", DefaultLogLevel)
	viper.SetDefault("artifacts", artifacts.DefaultArtifactsDir)

	// Set up cluster defaults
	viper.SetDefault("namespace", DefaultNamespace)
	viper.SetDefault("serviceaccount", DefaultServiceAccount)

	// Set up scorecard wait time default
	viper.SetDefault("scorecard_wait_time", DefaultScorecardWaitTime)
}

// preRunConfig is used by cobra.PreRun in all non-root commands to load all necessary configurations
func preRunConfig(cmd *cobra.Command, args []string) {
	// set up logging
	logname := viper.GetString("logfile")
	logFile, err := os.OpenFile(logname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err == nil {
		mw := io.MultiWriter(os.Stderr, logFile)
		log.SetOutput(mw)
	} else {
		log.Debug("Failed to log to file, using default stderr")
	}
	if ll, err := log.ParseLevel(viper.GetString("loglevel")); err == nil {
		log.SetLevel(ll)
	}

	log.SetFormatter(&log.TextFormatter{})
	if !configFileUsed {
		log.Debug("config file not found, proceeding without it")
	}
}
