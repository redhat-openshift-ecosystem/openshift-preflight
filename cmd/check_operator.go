package cmd

import (
	"fmt"
	"os"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"github.com/spf13/cobra"
)

var checkOperatorCmd = &cobra.Command{
	Use:   "operator",
	Short: "Run checks for an Operator",
	Long:  `This command will run the Certification checks for an Operator bundle image. `,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("%w: An operator image positional argument is required", errors.ErrInsufficientPosArguments)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Expect exactly one positional arg. Check here instead of using builtin Args key
		// so that we can get a more user-friendly error message

		operatorImage := args[0]

		cfg := runtime.Config{
			Image:          operatorImage,
			EnabledChecks:  engine.OperatorPolicy(),
			ResponseFormat: DefaultOutputFormat,
		}

		engine, err := engine.NewForConfig(cfg)
		if err != nil {
			return err
		}

		formatter, err := formatters.NewForConfig(cfg)
		if err != nil {
			return err
		}

		engine.ExecuteChecks()
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

func init() {
	checkOperatorCmd.SetUsageTemplate(
		`Usage:
  preflight check operator <url to Operator bundle image> [flags]
	
Flags:
  -h, --help   help for operator
`)
	checkCmd.AddCommand(checkOperatorCmd)
}
