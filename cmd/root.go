package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "preflight <container-image>",
	Short:   "Preflight Red Hat certification prep tool.",
	Version: version.Version.String(),
	Long: "A utility that allows you to pre-test your bundles, operators, and container before submitting for Red Hat Certification." +
		"\nChoose from any of the following checks:" +
		"\n\t" + strings.Join(engine.AllChecks(), ", ") +
		"\nChoose from any of the following output formats:" +
		"\n\t" + strings.Join(formatters.AllFormats(), ", "),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Expect exactly one positional arg. Check here instead of using builtin Args key
		// so that we can get a more user-friendly error message
		if len(args) != 1 {
			return fmt.Errorf("%w: A container image positional argument is required", errors.ErrInsufficientPosArguments)
		}
		containerImage := args[0]

		cfg := runtime.Config{
			Image:          containerImage,
			EnabledChecks:  parseEnabledChecksValue(),
			ResponseFormat: parseOutputFormat(),
		}

		engine, err := engine.NewForConfig(cfg)
		if err != nil {
			return err
		}

		formatter, err := formatters.NewForConfig(cfg)
		if err != nil {
			return err
		}

		engine.ExecuteChecks(logger)
		results := engine.Results()

		// return results to the user
		formattedResults, err := formatter.Format(results)
		if err != nil {
			return err
		}

		fmt.Fprint(os.Stdout, string(formattedResults))

		return nil
	},
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
