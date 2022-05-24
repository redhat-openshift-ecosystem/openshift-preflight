package cmd

import (
	"context"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Run checks for an operator or container",
	Long:  "This command will allow you to execute the Red Hat Certification tests for an operator or a container.",
}

func init() {
	checkCmd.PersistentFlags().BoolP("list-checks", "l", false, "lists all the checks run for a given check")

	checkCmd.PersistentFlags().StringP("docker-config", "d", "", "Path to docker config.json file. This value is optional for publicly accessible images.\n"+
		"However, it is strongly encouraged for public Docker Hub images,\n"+
		"due to the rate limit imposed for unauthenticated requests. (env: PFLT_DOCKERCONFIG)")
	viper.BindPFlag("dockerConfig", checkCmd.PersistentFlags().Lookup("docker-config"))

	checkCmd.PersistentFlags().String("artifacts", "", "Where check-specific artifacts will be written. (env: PFLT_ARTIFACTS)")
	viper.BindPFlag("artifacts", checkCmd.PersistentFlags().Lookup("artifacts"))

	rootCmd.AddCommand(checkCmd)
}

func writeJunitIfEnabled(ctx context.Context, results runtime.Results) error {
	if !viper.GetBool("junit") {
		return nil
	}

	var cfg runtime.Config
	cfg.ResponseFormat = "junitxml"

	junitformatter, err := formatters.NewForConfig(cfg)
	if err != nil {
		return err
	}
	junitResults, err := junitformatter.Format(ctx, results)
	if err != nil {
		return err
	}

	junitFilename, err := artifacts.WriteFile("results-junit.xml", string(junitResults))
	if err != nil {
		return err
	}
	log.Tracef("JUnitXML written to %s", junitFilename)

	return nil
}
