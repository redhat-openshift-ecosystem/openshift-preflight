// Package cmd implements the command-line interface for Preflight.
package cmd

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/viper"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"

	"github.com/bombsimon/logrusr/v4"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	spfviper "github.com/spf13/viper"
)

var configFileUsed bool

func init() {
	cobra.OnInitialize(func() { initConfig(viper.Instance()) })
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

	viper := viper.Instance()
	rootCmd.PersistentFlags().String("logfile", "", "Where the execution logfile will be written. (env: PFLT_LOGFILE)")
	_ = viper.BindPFlag("logfile", rootCmd.PersistentFlags().Lookup("logfile"))

	rootCmd.PersistentFlags().String("loglevel", "", "The verbosity of the preflight tool itself. Ex. warn, debug, trace, info, error. (env: PFLT_LOGLEVEL)")
	_ = viper.BindPFlag("loglevel", rootCmd.PersistentFlags().Lookup("loglevel"))

	rootCmd.AddCommand(checkCmd())
	rootCmd.AddCommand(listChecksCmd())
	rootCmd.AddCommand(runtimeAssetsCmd())
	rootCmd.AddCommand(supportCmd())
	rootCmd.AddCommand(experimentalCmd())

	return rootCmd
}

func Execute() error {
	return rootCmd().ExecuteContext(context.Background())
}

func initConfig(viper *spfviper.Viper) {
	// set up ENV var support
	viper.SetEnvPrefix("pflt")
	viper.AutomaticEnv()

	// set up optional config file support
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	configFileUsed = true
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(spfviper.ConfigFileNotFoundError); ok {
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
	viper := viper.Instance()
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{DisableColors: true})

	// set up logging
	logname := viper.GetString("logfile")
	logFile, err := os.OpenFile(logname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err == nil {
		mw := io.MultiWriter(os.Stderr, logFile)
		l.SetOutput(mw)
	} else {
		l.Infof("Failed to log to file, using default stderr")
	}
	if ll, err := logrus.ParseLevel(viper.GetString("loglevel")); err == nil {
		l.SetLevel(ll)
	}

	// if we are in the offline flow redirect log file to exist in the directory where all other artifact exist
	if viper.GetBool("offline") {
		// Get the base name of the logfile, in case logfile has a path
		baseLogName := filepath.Base(logname)
		artifacts := viper.GetString("artifacts")

		// ignoring error since OpenFile will error and we'll still have the multiwriter from above
		_ = os.Mkdir(artifacts, 0o777)

		artifactsLogFile, err := os.OpenFile(filepath.Join(artifacts, baseLogName), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err == nil {
			mw := io.MultiWriter(os.Stderr, logFile, artifactsLogFile)
			l.SetOutput(mw)
		}

		// setting log level to trace, to provide the most detailed logs possible
		l.SetLevel(logrus.TraceLevel)
	}

	if !configFileUsed {
		l.Debug("config file not found, proceeding without it")
	}

	logger := logrusr.New(l)
	ctx := logr.NewContext(cmd.Context(), logger)
	cmd.SetContext(ctx)
}
