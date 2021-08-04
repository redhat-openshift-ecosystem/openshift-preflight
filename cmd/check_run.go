package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var checkRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a single check inside of podman unshare",
	Long: `This command will run a check while wrapped inside of podman unshare.
It is an internal command, and is not meant to be called by the user.
It takes its input from environment variables only.`,
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Expect an environment variable named PREFLIGHT_CHECK_EXEC
		// It will specify the one check to execute inside of the podman unshare
		// environment.
		enabledCheck, ok := os.LookupEnv("PREFLIGHT_EXEC_CHECK")
		if !ok {
			log.Error("Enabled check envvar not specified")
			return errors.New("required environment variable PREFLIGHT_EXEC_CHECK not specified")
		}
		image, ok := os.LookupEnv("PREFLIGHT_EXEC_IMAGE")
		if !ok {
			log.Error("Operator image envvar not specified")
			return errors.New("required environment variable PREFLIGHT_EXEC_IMAGE not specified")
		}
		mounted, ok := os.LookupEnv("PREFLIGHT_EXEC_MOUNTED")
		if !ok {
			mounted = "false"
		}

		cfg := runtime.Config{
			Image:          image,
			EnabledChecks:  []string{enabledCheck},
			ResponseFormat: DefaultOutputFormat,
			Mounted:        mounted == "true",
		}

		engine, err := engine.NewForConfig(cfg)
		if err != nil {
			return err
		}

		formatter, err := formatters.NewForConfig(cfg)
		if err != nil {
			return err
		}

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

		fmt.Fprint(os.Stdout, string(formattedResults))

		return nil
	},
}

func init() {
	checkCmd.AddCommand(checkRunCmd)
}
