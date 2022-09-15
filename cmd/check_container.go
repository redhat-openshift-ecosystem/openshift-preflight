package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/policy"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
		PreRunE: validateCertificationProjectID,
		RunE:    checkContainerRunE,
	}

	checkContainerCmd.Flags().BoolVarP(&submit, "submit", "s", false, "submit check container results to red hat")
	_ = viper.BindPFlag("submit", checkContainerCmd.Flags().Lookup("submit"))

	checkContainerCmd.Flags().String("pyxis-api-token", "", "API token for Pyxis authentication (env: PFLT_PYXIS_API_TOKEN)")
	_ = viper.BindPFlag("pyxis_api_token", checkContainerCmd.Flags().Lookup("pyxis-api-token"))

	checkContainerCmd.Flags().String("pyxis-host", "", fmt.Sprintf("Host to use for Pyxis submissions. This will override Pyxis Env. Only set this if you know what you are doing.\n"+
		"If you do set it, it should include just the host, and the URI path. (env: PFLT_PYXIS_HOST)"))
	_ = viper.BindPFlag("pyxis_host", checkContainerCmd.Flags().Lookup("pyxis-host"))

	checkContainerCmd.Flags().String("pyxis-env", certification.DefaultPyxisEnv, "Env to use for Pyxis submissions.")
	_ = viper.BindPFlag("pyxis_env", checkContainerCmd.Flags().Lookup("pyxis-env"))

	checkContainerCmd.Flags().String("certification-project-id", "", fmt.Sprintf("Certification Project ID from connect.redhat.com/projects/{certification-project-id}/overview\n"+
		"URL paramater. This value may differ from the PID on the overview page. (env: PFLT_CERTIFICATION_PROJECT_ID)"))
	_ = viper.BindPFlag("certification_project_id", checkContainerCmd.Flags().Lookup("certification-project-id"))

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
	if certProject.Container.Type == "scratch" {
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

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Changed && strings.Contains(f.Value.String(), "--submit") {
			// We have --submit in one of the flags. That's a problem.
			// We will set the submit flag to true so that the next block functions properly
			submit = true
		}
	})

	// --submit was specified
	if submit {
		// If the flag is not marked as changed AND viper hasn't gotten it from environment, it's an error
		if !cmd.Flag("certification-project-id").Changed && !viper.IsSet("certification_project_id") {
			return fmt.Errorf("certification Project ID must be specified when --submit is present")
		}
		if !cmd.Flag("pyxis-api-token").Changed && !viper.IsSet("pyxis_api_token") {
			return fmt.Errorf("pyxis API Token must be specified when --submit is present")
		}

		// If the flag is marked as changed AND it's still empty, it's an error
		if cmd.Flag("certification-project-id").Changed && viper.GetString("certification_project_id") == "" {
			return fmt.Errorf("certification Project ID cannot be empty when --submit is present")
		}
		if cmd.Flag("pyxis-api-token").Changed && viper.GetString("pyxis_api_token") == "" {
			return fmt.Errorf("pyxis API Token cannot be empty when --submit is present")
		}

		// Finally, if either certification project id or pyxis api token start with '--', it's an error
		if strings.HasPrefix(viper.GetString("pyxis_api_token"), "--") || strings.HasPrefix(viper.GetString("certification_project_id"), "--") {
			return fmt.Errorf("pyxis API token and certification ID are required when --submit is present")
		}
	}

	return nil
}

// validateCertificationProjectID validates that the certification project id is in the proper format
// and throws an error if the value provided is in a legacy format that is not usable to query pyxis
func validateCertificationProjectID(cmd *cobra.Command, args []string) error {
	certificationProjectID := viper.GetString("certification_project_id")
	// splitting the certification project id into parts. if there are more than 2 elements in the array,
	// we know they inputted a legacy project id, which can not be used to query pyxis
	parts := strings.Split(certificationProjectID, "-")

	if len(parts) > 2 {
		return fmt.Errorf("certification project id: %s is improperly formatted see help command for instructions on obtaining proper value", certificationProjectID)
	}

	if parts[0] == "ospid" {
		viper.Set("certification_project_id", parts[1])
	}

	return nil
}
