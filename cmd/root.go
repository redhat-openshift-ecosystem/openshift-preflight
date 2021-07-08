package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "preflight",
	Short: "Preflight Red Hat certification prep tool.",
	Long:  "A utility that allows you to pre-test your bundles, operators, and container before submitting for Red Hat Certification.",
}

var (
	userEnabledChecks string
	userOutputFormat  string
)

func Execute() {
	// We don't set default values here because we want to parse the environment
	// in addition to the flags and enforce precedence between the two.
	rootCmd.Flags().StringVarP(&userEnabledChecks,
		"enabled-checks", "c", "", fmt.Sprintf(
			"Which checks to apply to the image to ensure compliance.\n(Env) %s",
			EnvEnabledChecks))
	rootCmd.Flags().StringVarP(&userOutputFormat,
		"output-format", "o", "", fmt.Sprintf(
			"The format for the check test results.\n(Env) %s (Default) %s",
			EnvOutputFormat, defaultOutputFormat))
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
