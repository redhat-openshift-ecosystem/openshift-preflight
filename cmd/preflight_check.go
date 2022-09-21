package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"

	log "github.com/sirupsen/logrus"
)

// preflightCheck executes checks, interacts with pyxis, format output, writes, and submits results.
func preflightCheck(
	ctx context.Context,
	cfg *runtime.Config,
	pc pyxisClient, //nolint:unparam // pyxisClient is currently unused.
	eng engine.CheckEngine,
	formatter formatters.ResponseFormatter,
	rw resultWriter,
	rs resultSubmitter,
) error {
	// configure the artifacts directory if the user requested a different directory.
	if cfg.Artifacts != "" {
		artifacts.SetDir(cfg.Artifacts)
	}

	// create the results file early to catch cases where we are not
	// able to write to the filesystem before we attempt to execute checks.
	resultsFilePath, err := artifacts.WriteFile(resultsFilenameWithExtension(formatter.FileExtension()), strings.NewReader(""))
	if err != nil {
		return err
	}
	resultsFile, err := rw.OpenFile(resultsFilePath)
	if err != nil {
		return err
	}
	defer resultsFile.Close()

	resultsOutputTarget := io.MultiWriter(os.Stdout, resultsFile)

	// execute the checks
	if err := eng.ExecuteChecks(ctx); err != nil {
		return err
	}
	results := eng.Results(ctx)

	// return results to the user and then close output files
	formattedResults, err := formatter.Format(ctx, results)
	if err != nil {
		return err
	}

	fmt.Fprintln(resultsOutputTarget, string(formattedResults))

	if cfg.WriteJUnit {
		if err := writeJUnit(ctx, results); err != nil {
			return err
		}
	}

	if cfg.Submit {
		if err := rs.Submit(ctx); err != nil {
			return err
		}
	}

	log.Infof("Preflight result: %s", convertPassedOverall(results.PassedOverall))

	return nil
}
