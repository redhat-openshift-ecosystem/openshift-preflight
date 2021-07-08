package cmd

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "preflight",
	Short:   "Preflight Red Hat certification prep tool.",
	Long:    "A utility that allows you to pre-test your bundles, operators, and container before submitting for Red Hat Certification.",
	Version: version.Version.String(),
}

func Execute() {
	rootCmd.Execute()
}
