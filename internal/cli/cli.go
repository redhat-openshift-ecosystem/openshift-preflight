package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/lib"
)

type CheckConfig struct {
	IncludeJUnitResults bool
	SubmitResults       bool
}

// RunPreflight executes checks, writes logs, results, and submits results if requested.
func RunPreflight(
	ctx context.Context,
	runChecks func(context.Context) (certification.Results, error),
	cfg CheckConfig,
	formatter formatters.ResponseFormatter,
	rw lib.ResultWriter,
	rs lib.ResultSubmitter,
) error {
	// Configure artifact writing if not already configured. For CLI
	// executions, we default to writing to the filesystem.
	artifactsWriter := artifacts.WriterFromContext(ctx)
	var err error
	if artifactsWriter == nil {
		return errors.New("no artifact writer was configured")
	}
	// Fail early if we cannot write to the results path.
	resultsFilePath, err := artifactsWriter.WriteFile(ResultsFilenameWithExtension(formatter.FileExtension()), strings.NewReader(""))
	if err != nil {
		return err
	}

	resultsFile, err := rw.OpenFile(resultsFilePath)
	if err != nil {
		return err
	}

	defer resultsFile.Close()
	resultsOutputTarget := io.MultiWriter(os.Stdout, resultsFile)

	// Execute Checks.
	results, err := runChecks(ctx)
	if err != nil {
		return err
	}

	// Format and write the results.
	formattedResults, err := formatter.Format(ctx, results)
	if err != nil {
		return err
	}

	fmt.Fprintln(resultsOutputTarget, string(formattedResults))

	// Optionally write the JUnit results alongside the regular results.
	if cfg.IncludeJUnitResults {
		if err := writeJUnit(ctx, results); err != nil {
			return err
		}
	}

	if cfg.SubmitResults {
		if err := rs.Submit(ctx); err != nil {
			return err
		}
	}

	log.Infof("Preflight result: %s", convertPassedOverall(results.PassedOverall))

	return nil
}

// writeJUnit will write JUnit results as an artifact using the ArtifactWriter configured
// in ctx.
func writeJUnit(ctx context.Context, results certification.Results) error {
	junitformatter, err := formatters.NewByName("junitxml")
	if err != nil {
		return err
	}

	junitResults, err := junitformatter.Format(ctx, results)
	if err != nil {
		return err
	}

	if aw := artifacts.WriterFromContext(ctx); aw != nil {
		junitFilename, err := aw.WriteFile("results-junit.xml", bytes.NewReader((junitResults)))
		if err != nil {
			return err
		}
		log.Tracef("JUnitXML written to %s", junitFilename)
	}

	return nil
}

func convertPassedOverall(passedOverall bool) string {
	if passedOverall {
		return "PASSED"
	}

	return "FAILED"
}

func ResultsFilenameWithExtension(ext string) string {
	return strings.Join([]string{"results", ext}, ".")
}
