package cmd

import (
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Run checks for an operator or container",
	Long:  "This command will allow you to execute the Red Hat Certification tests for an operator or a container.",
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
