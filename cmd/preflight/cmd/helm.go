package cmd

import (
	chartverifier "github.com/redhat-certification/chart-verifier/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// helmCmd contains chart-verifier related subcommands.
func helmCmd() *cobra.Command {
	helmCmd := &cobra.Command{
		Use:     "helm-chart",
		Short:   "Run helm chart certification tooling",
		Long:    "This command allows you to run certification workflows against your Helm charts",
		Aliases: []string{"chart-verifier"},
	}

	helmCmd.AddCommand(chartverifier.NewVerifyCmd(viper.New()))
	helmCmd.AddCommand(chartverifier.NewVersionCmd())
	helmCmd.AddCommand(chartverifier.NewReportCmd(viper.New()))

	return helmCmd
}
