package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var certifyCmd = &cobra.Command{
	Use:    "certify",
	Short:  "Submits check results to Red Hat",
	Long:   `This command will run all the checks for a container or operator and submit the results to Red Hat for Certification consideration `,
	PreRun: preRunConfig,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Certify command not implemented")
	},
}

func init() {
	rootCmd.AddCommand(certifyCmd)
}
