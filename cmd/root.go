// Package cmd implements the command-line interface for Preflight.
package cmd

import (
	"os"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:     "preflight",
	Short:   "Preflight Red Hat certification prep tool.",
	Long:    "A utility that allows you to pre-test your bundles, operators, and container before submitting for Red Hat Certification.",
	Version: version.Version.String(),
	Args:    cobra.MinimumNArgs(1),
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String("logfile", "", "Where the execution logfile will be written. (env: PFLT_LOGFILE)")
	viper.BindPFlag("logfile", rootCmd.PersistentFlags().Lookup("logfile"))

	rootCmd.PersistentFlags().String("loglevel", "", "The verbosity of the preflight tool itself. Ex. warn, debug, trace, info, error. (env: PFLT_LOGLEVEL)")
	viper.BindPFlag("loglevel", rootCmd.PersistentFlags().Lookup("loglevel"))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}
}
