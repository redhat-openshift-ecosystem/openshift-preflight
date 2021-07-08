package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var supportCmd = &cobra.Command{
	Use:   "support",
	Short: "Submits a support request",
	Long: `This command will submit a support request to Red Hat along with the logs from the latest Preflight checck.
	This command can be used when you'd like assistance from Red Hat Support when attempting to pass your certification checks. `,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Support command not implemented")
	},
}

func init() {
	rootCmd.AddCommand(supportCmd)
}
