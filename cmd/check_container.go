package cmd

import (
	"context"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/policy"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var submit bool

func checkContainerCmd() *cobra.Command {
	checkContainerCmd := &cobra.Command{
		Use:   "container",
		Short: "Run checks for a container",
		Long:  `This command will run the Certification checks for a container image. `,
		Args:  checkContainerPositionalArgs,
		// this fmt.Sprintf is in place to keep spacing consistent with cobras two spaces that's used in: Usage, Flags, etc
		Example: fmt.Sprintf("  %s", "preflight check container quay.io/repo-name/container-name:version"),
		RunE:    checkContainerRunE,
	}

	checkContainerCmd.Flags().BoolVarP(&submit, "submit", "s", false, "submit check container results to red hat")
	viper.BindPFlag("submit", checkContainerCmd.Flags().Lookup("submit"))

	checkContainerCmd.Flags().String("pyxis-api-token", "", "API token for Pyxis authentication (env: PFLT_PYXIS_API_TOKEN)")
	viper.BindPFlag("pyxis_api_token", checkContainerCmd.Flags().Lookup("pyxis-api-token"))

	checkContainerCmd.Flags().String("pyxis-host", "", fmt.Sprintf("Host to use for Pyxis submissions. This will override Pyxis Env. Only set this if you know what you are doing.\n"+
		"If you do set it, it should include just the host, and the URI path. (env: PFLT_PYXIS_HOST)"))
	viper.BindPFlag("pyxis_host", checkContainerCmd.Flags().Lookup("pyxis-host"))

	checkContainerCmd.Flags().String("pyxis-env", certification.DefaultPyxisEnv, "Env to use for Pyxis submissions.")
	viper.BindPFlag("pyxis_env", checkContainerCmd.Flags().Lookup("pyxis-env"))

	checkContainerCmd.Flags().String("certification-project-id", "", fmt.Sprintf("Certification Project ID from connect.redhat.com/projects/{certification-project-id}/overview\n"+
		"URL paramater. This value may differ from the PID on the overview page. (env: PFLT_CERTIFICATION_PROJECT_ID)"))
	viper.BindPFlag("certification_project_id", checkContainerCmd.Flags().Lookup("certification-project-id"))

	return checkContainerCmd
}

// checkContainerRunner contains all of the components necessary to run checkContainer.
type checkContainerRunner struct {
	cfg       *runtime.Config
	pc        pyxisClient
	eng       engine.CheckEngine
	formatter formatters.ResponseFormatter
	rw        resultWriter
	rs        resultSubmitter
}

func newCheckContainerRunner(ctx context.Context, cfg *runtime.Config) (*checkContainerRunner, error) {
	cfg.Policy = policy.PolicyContainer
	cfg.Submit = submit

	pyxisClient := newPyxisClient(ctx, cfg.ReadOnly())
	// If we have a pyxisClient, we can query for container policy exceptions.
	if pyxisClient != nil {
		policy, err := getContainerPolicyExceptions(ctx, pyxisClient)
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

	rs := resolveSubmitter(pyxisClient, cfg.ReadOnly())

	return &checkContainerRunner{
		cfg:       cfg,
		pc:        pyxisClient,
		eng:       engine,
		formatter: fmttr,
		rw:        &runtime.ResultWriterFile{},
		rs:        rs,
	}, nil
}

// checkContainerRunE executes checkContainer using the user args to inform the execution.
func checkContainerRunE(cmd *cobra.Command, args []string) error {
	log.Info("certification library version ", version.Version.String())
	ctx := cmd.Context()
	containerImage := args[0]

	// Render the Viper configuration as a runtime.Config
	cfg, err := runtime.NewConfigFrom(*viper.GetViper())
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	cfg.Image = containerImage
	cfg.ResponseFormat = formatters.DefaultFormat

	checkContainer, err := newCheckContainerRunner(ctx, cfg)
	if err != nil {
		return err
	}

	// Run the  container check.
	cmd.SilenceUsage = true
	return preflightCheck(ctx,
		checkContainer.cfg,
		checkContainer.pc,
		checkContainer.eng,
		checkContainer.formatter,
		checkContainer.rw,
		checkContainer.rs,
	)
}

// resolveSubmitter will build out a resultSubmitter if the provided pyxisClient, pc, is not nil.
// The pyxisClient is a required component of the submitter. If pc is nil, then a noop submitter
// is returned instead, which does nothing.
func resolveSubmitter(pc pyxisClient, cfg certification.Config) resultSubmitter {
	if pc != nil {
		return &containerCertificationSubmitter{
			certificationProjectID: cfg.CertificationProjectID(),
			pyxis:                  pc,
			dockerConfig:           cfg.DockerConfig(),
			preflightLogFile:       cfg.LogFile(),
		}
	}

	return &noopSubmitter{emitLog: true}
}

// getContainerPolicyExceptions will query Pyxis to determine if
// a given project has a certification excemptions, such as root or scratch.
// This will then return the corresponding policy.
//
// If no policy exception flags are found on the project, the standard
// container policy is returned.
func getContainerPolicyExceptions(ctx context.Context, pc pyxisClient) (policy.Policy, error) {
	certProject, err := pc.GetProject(ctx)
	if err != nil {
		return "", fmt.Errorf("could not retrieve project: %w", err)
	}
	log.Debugf("Certification project name is: %s", certProject.Name)
	if certProject.Container.OsContentType == "scratch" {
		return policy.PolicyScratch, nil
	}

	// if a partner sets `Host Level Access` in connect to `Privileged`, enable RootExceptionContainerPolicy checks
	if certProject.Container.Privileged {
		return policy.PolicyRoot, nil
	}
	return policy.PolicyContainer, nil
}

func checkContainerPositionalArgs(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("a container image positional argument is required")
	}

	if submit {
		if !viper.IsSet("certification_project_id") {
			cmd.MarkFlagRequired("certification-project-id")
		}

		if !viper.IsSet("pyxis_api_token") {
			cmd.MarkFlagRequired("pyxis-api-token")
		}
	}

	return nil
}
