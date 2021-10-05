package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"github.com/spf13/cobra"
)

var runtimeAssetsCmd = &cobra.Command{
	Use:    "runtime-assets",
	Short:  "Returns information about assets used at runtime.",
	Long:   `This command will return information on all runtime assets used by preflight. Useful for preparing a disconnected environment intending to utilize preflight.`,
	PreRun: preRunConfig,
	RunE: func(cmd *cobra.Command, args []string) error {
		assets := runtime.Assets()

		assetsJSON, err := json.MarshalIndent(assets, "", "    ")
		if err != nil {
			return err
		}

		fmt.Println(string(assetsJSON))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runtimeAssetsCmd)
}
