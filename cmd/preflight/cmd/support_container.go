package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func supportContainerCmd() *cobra.Command {
	supportContainer := &cobra.Command{
		Use:   "container <your project ID>",
		Short: "Creates a support request",
		Args:  cobra.ExactArgs(1),
		Long:  `Generate a URL that can be used to open a ticket with Red Hat Support if you're having an issue passing certification checks.`,
		RunE:  supportContainerRunE,
	}

	return supportContainer
}

func supportContainerRunE(cmd *cobra.Command, args []string) error {
	pid := args[0]
	// prurl is not needed for container support
	prurl := ""
	ptype := "container"

	support, err := newSupportTextGenerator(ptype, pid, prurl)
	if err != nil {
		return err
	}

	fmt.Fprint(cmd.OutOrStdout(), support.Generate())
	return nil
}
