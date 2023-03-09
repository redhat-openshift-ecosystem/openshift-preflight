package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// The root command tries to create the prelfight log early
			// even if we don't write to it. Set it to DevNull because
			// experimental won't make use of it.
			viper.Set("logfile", os.DevNull)
		},
	}

	// Hide preflight-specific global flags from being suggested to experimental commands,
	// where they're not expected to function in the same capacity.
	expCmd.SetHelpFunc(func(command *cobra.Command, strings []string) {
		_ = command.Parent().Flags().MarkHidden("logfile")
		_ = command.Parent().Flags().MarkHidden("loglevel")
		command.Root().HelpFunc()(command, strings)
	})

	expCmd.AddCommand(helmCmd())

	return expCmd
}
