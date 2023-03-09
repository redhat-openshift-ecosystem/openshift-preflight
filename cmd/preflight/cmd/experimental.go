package cmd

import (
	"github.com/spf13/cobra"
)

// experimentalCmd returns a Cobra command containing experimental subcommands.
// These subcommands are not supported for production workflows.
func experimentalCmd() *cobra.Command {
	expCmd := &cobra.Command{
		Use:     "experimental",
		Short:   "Run experimental commands within Preflight",
		Long:    "This command will allow you to run experimental subcommands of the Preflight tool. These commands are not supported, may be potentially buggy, and should not be used by most people outside of specific testing cases. Experimental subcommands are subject to change or removal without notice.",
		Hidden:  true,
		Aliases: []string{"exp"},
	}

	expCmd.AddCommand(helmCmd())

	return expCmd
}
