package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func supportOperatorCmd() *cobra.Command {
	supportOperator := &cobra.Command{
		Use:   "operator <your project ID> <pullRequestURL>",
		Short: "Creates a support request for an operator",
		Args:  cobra.ExactArgs(2),
		Long:  `Generate a URL that can be used to open a ticket with Red Hat Support if you're having an issue passing certification checks.`,
		RunE:  supportOperatorRunE,
	}

	return supportOperator
}

func supportOperatorRunE(cmd *cobra.Command, args []string) error {
	pid := args[0]
	prurl := args[1]
	ptype := "operator"

	support, err := newSupportTextGenerator(ptype, pid, prurl)
	if err != nil {
		return err
	}

	fmt.Fprint(cmd.OutOrStdout(), support.Generate())
	return nil
}
