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

var checkContainerCmd = &cobra.Command{
	Use:   "container",
	Short: "Run checks for a container",
	Long:  `This command will run the Certification checks for a container image. `,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("%w: A container image positional argument is required", errors.ErrInsufficientPosArguments)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Expect exactly one positional arg. Check here instead of using builtin Args key
		// so that we can get a more user-friendly error message

		containerImage := args[0]

		cfg := runtime.Config{
			Image:          containerImage,
			EnabledChecks:  engine.ContainerPolicy(),
			ResponseFormat: DefaultOutputFormat,
			LogFile:        DefaultLogFile,
		}

		if err := initLogger(cfg); err != nil {
			return fmt.Errorf("%w: %s", errors.ErrInitializingLogger, err)
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
	checkContainerCmd.SetUsageTemplate(
		`Usage:
  preflight check container <url to container image> [flags]
	
Flags:
  -h, --help   help for container
`)
	checkCmd.AddCommand(checkContainerCmd)
}
