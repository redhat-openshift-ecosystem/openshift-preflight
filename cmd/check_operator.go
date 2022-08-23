package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/policy"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func checkOperatorCmd() *cobra.Command {
	checkOperatorCmd := &cobra.Command{
		Use:   "operator",
		Short: "Run checks for an Operator",
		Long:  `This command will run the Certification checks for an Operator bundle image. `,
		Args:  checkOperatorPositionalArgs,
		// this fmt.Sprintf is in place to keep spacing consistent with cobras two spaces that's used in: Usage, Flags, etc
		Example: fmt.Sprintf("  %s", "preflight check operator quay.io/repo-name/operator-bundle:version"),
		RunE:    checkOperatorRunE,
	}
	checkOperatorCmd.Flags().String("namespace", "", "The namespace to use when running OperatorSDK Scorecard. (env: PFLT_NAMESPACE)")
	_ = viper.BindPFlag("namespace", checkOperatorCmd.Flags().Lookup("namespace"))

	checkOperatorCmd.Flags().String("serviceaccount", "", "The service account to use when running OperatorSDK Scorecard. (env: PFLT_SERVICEACCOUNT)")
	_ = viper.BindPFlag("serviceaccount", checkOperatorCmd.Flags().Lookup("serviceaccount"))

	checkOperatorCmd.Flags().String("scorecard-image", "", "A uri that points to the scorecard image digest, used in disconnected environments.\n"+
		"It should only be used in a disconnected environment. Use preflight runtime-assets on a connected \n"+
		"workstation to generate the digest that needs to be mirrored. (env: PFLT_SCORECARD_IMAGE)")
	_ = viper.BindPFlag("scorecard_image", checkOperatorCmd.Flags().Lookup("scorecard-image"))

	checkOperatorCmd.Flags().String("scorecard-wait-time", "", "A time value that will be passed to scorecard's --wait-time environment variable.\n"+
		"(env: PFLT_SCORECARD_WAIT_TIME)")
	_ = viper.BindPFlag("scorecard_wait_time", checkOperatorCmd.Flags().Lookup("scorecard-wait-time"))

	checkOperatorCmd.Flags().String("channel", "", "The name of the operator channel which is used by DeployableByOLM to deploy the operator.\n"+
		"If empty, the default operator channel in bundle's annotations file is used.. (env: PFLT_CHANNEL)")
	_ = viper.BindPFlag("channel", checkOperatorCmd.Flags().Lookup("channel"))

	return checkOperatorCmd
}

// checkOperatorRunner contains all of the components necessary to run checkOperator.
type checkOperatorRunner struct {
	cfg       *runtime.Config
	eng       engine.CheckEngine
	formatter formatters.ResponseFormatter
	rw        resultWriter
}

// newCheckOperatorRunner returns a checkOperatorRunner containing all of the tooling necessary
// to run checkOperator.
func newCheckOperatorRunner(ctx context.Context, cfg *runtime.Config) (*checkOperatorRunner, error) {
	cfg.Policy = policy.PolicyOperator
	cfg.Submit = false // there's no such thing as submitting for operators today.

	engine, err := engine.NewForConfig(ctx, cfg.ReadOnly())
	if err != nil {
		return nil, err
	}

	fmttr, err := formatters.NewForConfig(cfg.ReadOnly())
	if err != nil {
		return nil, err
	}

	return &checkOperatorRunner{
		cfg:       cfg,
		eng:       engine,
		formatter: fmttr,
		rw:        &runtime.ResultWriterFile{},
	}, nil
}

// ensureKubeconfigIsSet ensures that the KUBECONFIG environment variable has a value.
func ensureKubeconfigIsSet() error {
	if _, ok := os.LookupEnv("KUBECONFIG"); !ok {
		return fmt.Errorf("environment variable KUBECONFIG could not be found")
	}

	return nil
}

// ensureIndexImageConfigIsSet ensures that the PFLT_INDEXIMAGE environment variable has
// a value.
func ensureIndexImageConfigIsSet() error {
	if catalogImage := viper.GetString("indexImage"); len(catalogImage) == 0 {
		return fmt.Errorf("environment variable PFLT_INDEXIMAGE could not be found")
	}

	return nil
}

// checkOperatorRunE executes checkOperator using the user args to inform the execution.
func checkOperatorRunE(cmd *cobra.Command, args []string) error {
	log.Info("certification library version ", version.Version.String())
	ctx := cmd.Context()
	operatorImage := args[0]

	// Render the Viper configuration as a runtime.Config
	cfg, err := runtime.NewConfigFrom(*viper.GetViper())
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	cfg.Image = operatorImage
	cfg.ResponseFormat = formatters.DefaultFormat
	cfg.Bundle = true
	cfg.Scratch = true

	checkOperator, err := newCheckOperatorRunner(ctx, cfg)
	if err != nil {
		return err
	}

	// Run the operator check
	cmd.SilenceUsage = true
	return preflightCheck(ctx,
		checkOperator.cfg,
		nil, // no pyxisClient is necessary
		checkOperator.eng,
		checkOperator.formatter,
		checkOperator.rw,
		&noopSubmitter{}, // we do not submit these results.
	)
}

func checkOperatorPositionalArgs(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("an operator bundle image positional argument is required")
	}

	if err := ensureKubeconfigIsSet(); err != nil {
		return err
	}

	if err := ensureIndexImageConfigIsSet(); err != nil {
		return err
	}

	return nil
}
