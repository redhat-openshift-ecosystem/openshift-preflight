package lib

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/policy"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
)

// CheckContainerRunner contains all of the components necessary to run checkContainer.
type CheckContainerRunner struct {
	Cfg       *runtime.Config
	Pc        PyxisClient
	Eng       engine.CheckEngine
	Formatter formatters.ResponseFormatter
	Rw        ResultWriter
	Rs        ResultSubmitter
}

func NewCheckContainerRunner(ctx context.Context, cfg *runtime.Config, submit bool) (*CheckContainerRunner, error) {
	cfg.Policy = policy.PolicyContainer
	cfg.Submit = submit

	pyxisClient := NewPyxisClient(ctx, cfg.ReadOnly())
	// If we have a pyxisClient, we can query for container policy exceptions.
	if pyxisClient != nil {
		policy, err := GetContainerPolicyExceptions(ctx, pyxisClient)
		if err != nil {
			return nil, err
		}

		cfg.Policy = policy
	}

	engine, err := engine.NewForConfig(ctx, cfg.ReadOnly())
	if err != nil {
		return nil, err
	}

	fmttr, err := formatters.NewForConfig(cfg.ReadOnly())
	if err != nil {
		return nil, err
	}

	rs := ResolveSubmitter(pyxisClient, cfg.ReadOnly())

	return &CheckContainerRunner{
		Cfg:       cfg,
		Pc:        pyxisClient,
		Eng:       engine,
		Formatter: fmttr,
		Rw:        &runtime.ResultWriterFile{},
		Rs:        rs,
	}, nil
}

// CheckOperatorRunner contains all of the components necessary to run checkOperator.
type CheckOperatorRunner struct {
	Cfg       *runtime.Config
	Eng       engine.CheckEngine
	Formatter formatters.ResponseFormatter
	Rw        ResultWriter
}

// NewCheckOperatorRunner returns a CheckOperatorRunner containing all of the tooling necessary
// to run checkOperator.
func NewCheckOperatorRunner(ctx context.Context, cfg *runtime.Config) (*CheckOperatorRunner, error) {
	cfg.Policy = policy.PolicyOperator
	cfg.Submit = false // there's no such thing as submitting for operators today.

	engine, err := engine.NewForConfig(ctx, cfg.ReadOnly())
	if err != nil {
		return nil, err
	}

	fmttr, err := formatters.NewForConfig(cfg.ReadOnly())
	if err != nil {
		return nil, err
	}

	return &CheckOperatorRunner{
		Cfg:       cfg,
		Eng:       engine,
		Formatter: fmttr,
		Rw:        &runtime.ResultWriterFile{},
	}, nil
}

// ResolveSubmitter will build out a ResultSubmitter if the provided pyxisClient, pc, is not nil.
// The pyxisClient is a required component of the submitter. If pc is nil, then a noop submitter
// is returned instead, which does nothing.
func ResolveSubmitter(pc PyxisClient, cfg certification.Config) ResultSubmitter {
	if pc != nil {
		return &ContainerCertificationSubmitter{
			CertificationProjectID: cfg.CertificationProjectID(),
			Pyxis:                  pc,
			DockerConfig:           cfg.DockerConfig(),
			PreflightLogFile:       cfg.LogFile(),
		}
	}
	return NewNoopSubmitter(true, nil)
}

// GetContainerPolicyExceptions will query Pyxis to determine if
// a given project has a certification excemptions, such as root or scratch.
// This will then return the corresponding policy.
//
// If no policy exception flags are found on the project, the standard
// container policy is returned.
func GetContainerPolicyExceptions(ctx context.Context, pc PyxisClient) (policy.Policy, error) {
	certProject, err := pc.GetProject(ctx)
	if err != nil {
		return "", fmt.Errorf("could not retrieve project: %w", err)
	}
	// log.Debugf("Certification project name is: %s", certProject.Name)
	if certProject.Container.Type == "scratch" {
		if certProject.Container.Privileged {
			return policy.PolicyScratchRoot, nil
		}
		return policy.PolicyScratchNonRoot, nil
	}

	// if a partner sets `Host Level Access` in connect to `Privileged`, enable RootExceptionContainerPolicy checks
	if certProject.Container.Privileged {
		return policy.PolicyRoot, nil
	}
	return policy.PolicyContainer, nil
}

// PreflightCheck executes checks, interacts with pyxis, format output, writes, and submits results.
func PreflightCheck(
	ctx context.Context,
	cfg *runtime.Config,
	pc PyxisClient, //nolint:unparam // PyxisClient is currently unused.
	eng engine.CheckEngine,
	formatter formatters.ResponseFormatter,
	rw ResultWriter,
	rs ResultSubmitter,
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

func writeJUnit(ctx context.Context, results runtime.Results) error {
	var cfg runtime.Config
	cfg.ResponseFormat = "junitxml"

	junitformatter, err := formatters.NewForConfig(cfg.ReadOnly())
	if err != nil {
		return err
	}
	junitResults, err := junitformatter.Format(ctx, results)
	if err != nil {
		return err
	}

	junitFilename, err := artifacts.WriteFile("results-junit.xml", bytes.NewReader((junitResults)))
	if err != nil {
		return err
	}
	log.Tracef("JUnitXML written to %s", junitFilename)

	return nil
}

func resultsFilenameWithExtension(ext string) string {
	return strings.Join([]string{"results", ext}, ".")
}

func convertPassedOverall(passedOverall bool) string {
	if passedOverall {
		return "PASSED"
	}

	return "FAILED"
}
