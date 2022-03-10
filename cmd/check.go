package cmd

import (
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
	cobra.OnInitialize(initConfig)

	checkCmd.PersistentFlags().BoolP("list-checks", "l", false, "lists all the checks run for a given check")

	rootCmd.AddCommand(checkCmd)
}

func writeJunitIfEnabled(results runtime.Results) error {
	if !viper.GetBool("junit") {
		return nil
	}

	var cfg runtime.Config
	cfg.ResponseFormat = "junitxml"

	junitformatter, err := formatters.NewForConfig(cfg)
	if err != nil {
		return err
	}
	junitResults, err := junitformatter.Format(results)
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
