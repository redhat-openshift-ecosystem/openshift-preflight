package cmd

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func checkCmd() *cobra.Command {
	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Run checks for an operator or container",
		Long:  "This command will allow you to execute the Red Hat Certification tests for an operator or a container.",
	}

	checkCmd.PersistentFlags().StringP("docker-config", "d", "", "Path to docker config.json file. This value is optional for publicly accessible images.\n"+
		"However, it is strongly encouraged for public Docker Hub images,\n"+
		"due to the rate limit imposed for unauthenticated requests. (env: PFLT_DOCKERCONFIG)")
	_ = viper.BindPFlag("dockerConfig", checkCmd.PersistentFlags().Lookup("docker-config"))

	checkCmd.PersistentFlags().String("artifacts", "", "Where check-specific artifacts will be written. (env: PFLT_ARTIFACTS)")
	_ = viper.BindPFlag("artifacts", checkCmd.PersistentFlags().Lookup("artifacts"))

	checkCmd.AddCommand(checkContainerCmd())
	checkCmd.AddCommand(checkOperatorCmd())

	return checkCmd
}

// writeJUnit will write results as JUnit XML using the built-in formatter.
func writeJUnit(ctx context.Context, results runtime.Results) error {
	var cfg runtime.Config
	cfg.ResponseFormat = "junitxml"

	junitformatter, err := formatters.NewForConfig(cfg.ReadOnly())
	if err != nil {
		return err
	}
	junitResults, err := junitformatter.Format(ctx, results)
	if err != nil {
		return err
	}

	junitFilename, err := artifacts.WriteFile("results-junit.xml", bytes.NewReader((junitResults)))
	if err != nil {
		return err
	}
	log.Tracef("JUnitXML written to %s", junitFilename)

	return nil
}

func resultsFilenameWithExtension(ext string) string {
	return strings.Join([]string{"results", ext}, ".")
}

func buildConnectURL(projectID string) string {
	connectURL := fmt.Sprintf("https://connect.redhat.com/projects/%s", projectID)

	pyxisEnv := viper.GetString("pyxis_env")
	if len(pyxisEnv) > 0 && pyxisEnv != "prod" {
		connectURL = fmt.Sprintf("https://connect.%s.redhat.com/projects/%s", viper.GetString("pyxis_env"), projectID)
	}

	return connectURL
}

func buildOverviewURL(projectID string) string {
	return fmt.Sprintf("%s/overview", buildConnectURL(projectID))
}

func buildScanResultsURL(projectID string, imageID string) string {
	return fmt.Sprintf("%s/images/%s/scan-results", buildConnectURL(projectID), imageID)
}

func convertPassedOverall(passedOverall bool) string {
	if passedOverall {
		return "PASSED"
	}

	return "FAILED"
}
