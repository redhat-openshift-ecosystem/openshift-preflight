package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"github.com/spf13/cobra"
)

var oldCheckOperatorCmd = &cobra.Command{
	Use:   "operator-old",
	Short: "Run checks for an Operator using the previous engine",
	Long:  `This command will run the Certification checks for an Operator bundle image. `,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("%w: An operator image positional argument is required", errors.ErrInsufficientPosArguments)
		}
		return nil
	},
	PreRun: preRunConfig,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Expect exactly one positional arg. Check here instead of using builtin Args key
		// so that we can get a more user-friendly error message

		operatorImage := args[0]

		if _, ok := os.LookupEnv("KUBECONFIG"); !ok {
			return errors.ErrNoKubeconfig
		}

		cfg := runtime.Config{
			Image:          operatorImage,
			EnabledChecks:  engine.OldOperatorPolicy(),
			ResponseFormat: DefaultOutputFormat,
			Bundle:         true,
		}

		engine, err := engine.NewShellEngineForConfig(cfg)
		if err != nil {
			return err
		}

		formatter, err := formatters.NewForConfig(cfg)
		if err != nil {
			return err
		}

		// create the results file early to catch cases where we are not
		// able to write to the filesystem before we attempt to execute checks.
		resultsFile, err := os.OpenFile(
			resultsFilenameWithExtension(formatter.FileExtension()),
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
			0600,
		)

		if err != nil {
			return err
		}

		// also write to stdout
		resultsOutputTarget := io.MultiWriter(os.Stdout, resultsFile)

		// execute the checks
		if err := engine.ExecuteChecks(); err != nil {
			return err
		}
		results := engine.Results()

		// return results to the user and then close output files
		formattedResults, err := formatter.Format(results)
		if err != nil {
			return err
		}

		fmt.Fprint(resultsOutputTarget, string(formattedResults))
		if err := resultsFile.Close(); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	checks := strings.Join(engine.OldOperatorPolicy(), "\n- ")

	usage := "\n" + `The checks that will be executed are the following:` + "\n- " +
		checks + "\n\n" +
		`Usage:
  preflight check operator <url to Operator bundle image> [flags]
	
Flags:
  -h, --help   help for operator
`
	checkOperatorCmd.SetUsageTemplate(usage)
	checkCmd.AddCommand(oldCheckOperatorCmd)
}
