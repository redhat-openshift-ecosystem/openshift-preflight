package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var checkOperatorCmd = &cobra.Command{
	Use:   "operator",
	Short: "Run checks for an Operator",
	Long:  `This command will run the Certification checks for an Operator bundle image. `,
	Args: func(cmd *cobra.Command, args []string) error {
		if l, _ := cmd.Flags().GetBool("list-checks"); l {
			fmt.Printf("\n%s\n%s%s\n", "The checks that will be executed are the following:", "- ",
				strings.Join(engine.OperatorPolicy(), "\n- "))

			// exiting gracefully instead of retuning, otherwise cobra calls RunE
			os.Exit(0)
		}

		if len(args) != 1 {
			return fmt.Errorf("%w: An operator image positional argument is required", errors.ErrInsufficientPosArguments)
		}
		return nil
	},
	// this fmt.Sprintf is in place to keep spacing consistent with cobras two spaces that's used in: Usage, Flags, etc
	Example: fmt.Sprintf("  %s", "preflight check operator quay.io/repo-name/operator-bundle:version"),
	PreRun:  preRunConfig,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Expect exactly one positional arg. Check here instead of using builtin Args key
		// so that we can get a more user-friendly error message

		log.Info("certification library version ", version.Version.String())

		ctx := context.Background()

		operatorImage := args[0]

		if _, ok := os.LookupEnv("KUBECONFIG"); !ok {
			return errors.ErrNoKubeconfig
		}

		if catalogImage := viper.GetString("indexImage"); len(catalogImage) == 0 {
			return errors.ErrIndexImageUndefined
		}

		cfg := runtime.Config{
			Image:          operatorImage,
			EnabledChecks:  engine.OperatorPolicy(),
			ResponseFormat: DefaultOutputFormat,
			Bundle:         true,
			Scratch:        true,
		}

		engine, err := engine.NewForConfig(cfg)
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
			filepath.Join(artifacts.Path(), resultsFilenameWithExtension(formatter.FileExtension())),
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
			0o600,
		)
		if err != nil {
			return err
		}

		// also write to stdout
		resultsOutputTarget := io.MultiWriter(os.Stdout, resultsFile)

		// At this point, we would no longer want usage information printed out
		// on error, so it doesn't contaminate the output.
		cmd.SilenceUsage = true

		// execute the checks
		if err := engine.ExecuteChecks(ctx); err != nil {
			return err
		}
		results := engine.Results(ctx)

		// return results to the user and then close output files
		formattedResults, err := formatter.Format(ctx, results)
		if err != nil {
			return err
		}

		fmt.Fprint(resultsOutputTarget, string(formattedResults))
		if err := resultsFile.Close(); err != nil {
			return err
		}

		if err := writeJunitIfEnabled(ctx, results); err != nil {
			return err
		}

		log.Infof("Preflight result: %s", convertPassedOverall(results.PassedOverall))

		return nil
	},
}

func init() {
	checkCmd.AddCommand(checkOperatorCmd)
}
