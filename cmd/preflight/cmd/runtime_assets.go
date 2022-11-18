package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/runtime"

	"github.com/spf13/cobra"
)

func runtimeAssetsCmd() *cobra.Command {
	runtimeAssetsCmd := &cobra.Command{
		Use:   "runtime-assets",
		Short: "Returns information about assets used at runtime.",
		Long:  `This command will return information on all runtime assets used by preflight. Useful for preparing a disconnected environment intending to utilize preflight.`,
		RunE:  runtimeAssetsRunE,
	}

	return runtimeAssetsCmd
}

func runtimeAssetsRunE(cmd *cobra.Command, args []string) error {
	if err := printAssets(cmd.Context(), cmd.OutOrStdout()); err != nil {
		return err
	}

	return nil
}

func printAssets(ctx context.Context, w io.Writer) error {
	assets := runtime.Assets(ctx)

	assetsJSON, err := prettyPrintJSON(assets)
	if err != nil {
		return err
	}

	fmt.Fprintln(w, assetsJSON)
	return nil
}

// prettyPrintJSON marhals v with standard pretty print spacing and returns
// it in string form.
func prettyPrintJSON(v interface{}) (string, error) {
	json, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		return "", err
	}

	return string(json), nil
}
