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
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/policy"
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
			return fmt.Errorf("an operator image positional argument is required")
		}
		return nil
	},
	// this fmt.Sprintf is in place to keep spacing consistent with cobras two spaces that's used in: Usage, Flags, etc
	Example: fmt.Sprintf("  %s", "preflight check operator quay.io/repo-name/operator-bundle:version"),
	RunE:    checkOperatorRunE,
}

// checkOperatorRunE is a cobra RunE compatible function that prepares
// the user configuration for check operator.
func checkOperatorRunE(cmd *cobra.Command, args []string) error {
	log.Info("certification library version ", version.Version.String())

	ctx := cmd.Context()

	operatorImage := args[0]

	if _, ok := os.LookupEnv("KUBECONFIG"); !ok {
		return fmt.Errorf("environment variable KUBECONFIG could not be found")
	}

	if catalogImage := viper.GetString("indexImage"); len(catalogImage) == 0 {
		return fmt.Errorf("environment variable PFLT_INDEXIMAGE could not be found")
	}

	cfg, err := runtime.NewConfigFrom(*viper.GetViper())
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	cfg.Image = operatorImage
	cfg.ResponseFormat = DefaultOutputFormat
	cfg.Bundle = true
	cfg.Scratch = true

	// Run the operator check
	cmd.SilenceUsage = true
	return checkOperator(ctx, cfg)
}

func checkOperator(ctx context.Context, cfg *runtime.Config) error {
	cfg.Policy = policy.PolicyOperator

	// configure the artifacts directory if the user requested a different directory.
	if cfg.Artifacts != "" {
		artifacts.SetDir(cfg.Artifacts)
	}

	engine, err := engine.NewForConfig(ctx, cfg.ReadOnly())
	if err != nil {
		return err
	}

	formatter, err := formatters.NewForConfig(cfg.ReadOnly())
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

	if cfg.WriteJUnit {
		if err := writeJUnit(ctx, results); err != nil {
			return err
		}
	}

	log.Infof("Preflight result: %s", convertPassedOverall(results.PassedOverall))

	return nil
}

func init() {
	checkOperatorCmd.Flags().String("namespace", "", "The namespace to use when running OperatorSDK Scorecard. (env: PFLT_NAMESPACE)")
	viper.BindPFlag("namespace", checkOperatorCmd.Flags().Lookup("namespace"))

	checkOperatorCmd.Flags().String("serviceaccount", "", "The service account to use when running OperatorSDK Scorecard. (env: PFLT_SERVICEACCOUNT)")
	viper.BindPFlag("serviceaccount", checkOperatorCmd.Flags().Lookup("serviceaccount"))

	checkOperatorCmd.Flags().String("scorecard-image", "", "A uri that points to the scorecard image digest, used in disconnected environments.\n"+
		"It should only be used in a disconnected environment. Use preflight runtime-assets on a connected \n"+
		"workstation to generate the digest that needs to be mirrored. (env: PFLT_SCORECARD_IMAGE)")
	viper.BindPFlag("scorecard_image", checkOperatorCmd.Flags().Lookup("scorecard-image"))

	checkOperatorCmd.Flags().String("scorecard-wait-time", "", "A time value that will be passed to scorecard's --wait-time environment variable.\n"+
		"(env: PFLT_SCORECARD_WAIT_TIME)")
	viper.BindPFlag("scorecard_wait_time", checkOperatorCmd.Flags().Lookup("scorecard-wait-time"))

	checkOperatorCmd.Flags().String("channel", "", "The name of the operator channel which is used by DeployableByOLM to deploy the operator.\n"+
		"If empty, the default operator channel in bundle's annotations file is used.. (env: PFLT_CHANNEL)")
	viper.BindPFlag("channel", checkOperatorCmd.Flags().Lookup("channel"))

	checkCmd.AddCommand(checkOperatorCmd)
}
