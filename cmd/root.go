// Package cmd implements the command-line interface for Preflight.
package cmd

import (
	"os"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "preflight",
	Short:   "Preflight Red Hat certification prep tool.",
	Long:    "A utility that allows you to pre-test your bundles, operators, and container before submitting for Red Hat Certification.",
	Version: version.Version.String(),
	Args:    cobra.MinimumNArgs(1),
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}
}
