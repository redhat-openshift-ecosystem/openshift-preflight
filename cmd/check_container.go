package cmd

import (
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

var checkContainerCmd = &cobra.Command{
	Use:   "container",
	Short: "Run checks for a container",
	Long:  `This command will run the Certification checks for a container image. `,
	Args: func(cmd *cobra.Command, args []string) error {
		if l, _ := cmd.Flags().GetBool("list-checks"); l {
			fmt.Println(fmt.Sprintf("\n%s\n%s%s", "The checks that will be executed are the following:", "- ",
				strings.Join(engine.ContainerPolicy(), "\n- ")))

			// exiting gracefully instead of retuning, otherwise cobra calls RunE
			os.Exit(0)
		}

		if len(args) != 1 {
			return fmt.Errorf("%w: A container image positional argument is required", errors.ErrInsufficientPosArguments)
		}
		return nil
	},
	// this fmt.Sprintf is in place to keep spacing consistent with cobras two spaces that's used in: Usage, Flags, etc
	Example: fmt.Sprintf("  %s", "preflight check container quay.io/repo-name/container-name:version"),
	PreRun:  preRunConfig,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Expect exactly one positional arg. Check here instead of using builtin Args key
		// so that we can get a more user-friendly error message

		log.Info("certification library version ", version.Version.String())

		containerImage := args[0]

		cfg := runtime.Config{
			Image:          containerImage,
			EnabledChecks:  engine.ContainerPolicy(),
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

		// create the results file early to catch cases where we are not
		// able to write to the filesystem before we attempt to execute checks.
		resultsFile, err := os.OpenFile(
			filepath.Join(artifacts.Path(), resultsFilenameWithExtension(formatter.FileExtension())),
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

		if err := writeJunitIfEnabled(results); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	checkContainerCmd.Flags().String("pyxis-api-token", "", "API token for Pyxis authentication")
	viper.BindPFlag("pyxis_api_token", checkContainerCmd.Flags().Lookup("pyxis-api-token"))

	checkCmd.AddCommand(checkContainerCmd)
}
