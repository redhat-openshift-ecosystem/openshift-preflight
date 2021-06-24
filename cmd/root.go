package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/komish/preflight/certification/engine"
	"github.com/komish/preflight/certification/errors"
	"github.com/komish/preflight/certification/formatters"
	"github.com/komish/preflight/certification/runtime"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "preflight <container-image>",
	Short: "Preflight Red Hat certification prep tool.",
	Long: "A utility that allows you to pre-test your bundles, operators, and container before submitting for Red Hat Certification." +
		"\nChoose from any of the following policies:" +
		"\n\t" + strings.Join(engine.AllPolicies(), ", ") +
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
			Image:           containerImage,
			EnabledPolicies: parseEnabledPoliciesValue(),
			ResponseFormat:  parseOutputFormat(),
		}

		engine, err := engine.NewForConfig(cfg)
		if err != nil {
			return err
		}

		formatter, err := formatters.NewForConfig(cfg)
		if err != nil {
			return err
		}

		engine.ExecutePolicies(logger)
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
	userEnabledPolicies string
	userOutputFormat    string
)

func Execute() {
	// We don't set default values here because we want to parse the environment
	// in addition to the flags and enforce precedence between the two.
	rootCmd.Flags().StringVarP(&userEnabledPolicies,
		"enabled-policies", "p", "", fmt.Sprintf(
			"Which policies to apply to the bundle to ensure compliance.\n(Env) %s",
			EnvEnabledPolicies))
	rootCmd.Flags().StringVarP(&userOutputFormat,
		"output-format", "o", "", fmt.Sprintf(
			"The format for the policy test results.\n(Env) %s (Default) %s",
			EnvOutputFormat, defaultOutputFormat))
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
