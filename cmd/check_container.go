package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/pyxis"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var submit bool

var checkContainerCmd = &cobra.Command{
	Use:   "container",
	Short: "Run checks for a container",
	Long:  `This command will run the Certification checks for a container image. `,
	Args: func(cmd *cobra.Command, args []string) error {
		if l, _ := cmd.Flags().GetBool("list-checks"); l {
			fmt.Printf("\n%s\n%s%s\n", "The checks that will be executed are the following:", "- ",
				strings.Join(engine.ContainerPolicy(), "\n- "))

			// exiting gracefully instead of retuning, otherwise cobra calls RunE
			os.Exit(0)
		}

		if len(args) != 1 {
			return fmt.Errorf("%w: A container image positional argument is required", errors.ErrInsufficientPosArguments)
		}

		if submit {
			if !viper.IsSet("dockerConfig") {
				cmd.MarkFlagRequired("docker-config")
			}

			if !viper.IsSet("certification_project_id") {
				cmd.MarkFlagRequired("certification-project-id")
			}

			if !viper.IsSet("pyxis_api_token") {
				cmd.MarkFlagRequired("pyxis-api-token")
			}
		}

		return nil
	},
	// this fmt.Sprintf is in place to keep spacing consistent with cobras two spaces that's used in: Usage, Flags, etc
	Example: fmt.Sprintf("  %s", "preflight check container quay.io/repo-name/container-name:version"),
	PreRun:  preRunConfig,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("certification library version ", version.Version.String())
		ctx := context.Background()
		containerImage := args[0]

		cfg := runtime.Config{
			Image:          containerImage,
			EnabledChecks:  engine.ContainerPolicy(),
			ResponseFormat: DefaultOutputFormat,
		}

		pyxisHost := pyxisHostLookup(viper.GetString("pyxis_env"), viper.GetString("pyxis_host"))

		projectId := viper.GetString("certification_project_id")
		if projectId != "" {
			if strings.HasPrefix(projectId, "ospid-") {
				projectId = strings.Split(projectId, "-")[1]
				// Since we want the modified version, write it back
				// to viper so that subsequent calls don't need to check
				viper.Set("certification_project_id", projectId)
			}
			apiToken := viper.GetString("pyxis_api_token")
			pyxisClient := pyxis.NewPyxisClient(pyxisHost, apiToken, projectId, &http.Client{Timeout: 60 * time.Second})
			certProject, err := pyxisClient.GetProject(ctx)
			if err != nil {
				log.Error(fmt.Errorf("%w: %s", errors.ErrRetrievingProject, err))
				return err
			}
			log.Debugf("Certification project name is: %s", certProject.Name)
			if certProject.Container.OsContentType == "scratch" {
				cfg.EnabledChecks = engine.ScratchContainerPolicy()
				cfg.Scratch = true
			}
		}

		engine, err := engine.NewForConfig(cfg)
		if err != nil {
			return err
		}

		formatter, err := formatters.NewForConfig(cfg)
		if err != nil {
			return err
		}

		// create the results file early to catch cases where we are not
		// able to write to the filesystem before we attempt to execute checks.
		resultsFile, err := os.OpenFile(
			filepath.Join(artifacts.Path(), resultsFilenameWithExtension(formatter.FileExtension())),
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
			0o600,
		)
		if err != nil {
			return err
		}

		// also write to stdout
		resultsOutputTarget := io.MultiWriter(os.Stdout, resultsFile)

		// At this point, we would no longer want usage information printed out
		// on error, so it doesn't contaminate the output.
		cmd.SilenceUsage = true

		// execute the checks
		if err := engine.ExecuteChecks(ctx); err != nil {
			return err
		}
		results := engine.Results(ctx)

		// return results to the user and then close output files
		formattedResults, err := formatter.Format(ctx, results)
		if err != nil {
			return err
		}

		fmt.Fprintln(resultsOutputTarget, string(formattedResults))
		if err := resultsFile.Close(); err != nil {
			return err
		}

		if err := writeJunitIfEnabled(ctx, results); err != nil {
			return err
		}

		// assemble artifacts and submit results to pyxis if user provided the submit flag.
		if submit {
			log.Info("preparing results that will be submitted to Red Hat")

			// you must provide a project ID in order to submit.
			if projectId == "" {
				return errors.ErrEmptyProjectID
			}

			// establish a pyxis client.
			apiToken := viper.GetString("pyxis_api_token")
			pyxisClient := pyxis.NewPyxisClient(viper.GetString("pyxis_host"), apiToken, projectId, &http.Client{Timeout: 60 * time.Second})

			// get the project info from pyxis
			certProject, err := pyxisClient.GetProject(ctx)
			if err != nil {
				log.Error(fmt.Errorf("%w: %s", errors.ErrRetrievingProject, err))
				return err
			}
			log.Tracef("CertProject: %+v", certProject)

			// read the provided docker config
			dockerConfigJsonFile, err := os.Open(viper.GetString("dockerConfig"))
			if err != nil {
				return err
			}
			defer dockerConfigJsonFile.Close()

			dockerConfigJsonBytes, err := io.ReadAll(dockerConfigJsonFile)
			if err != nil {
				return err
			}

			certProject.Container.DockerConfigJSON = string(dockerConfigJsonBytes)

			// prepare submission
			submission, err := pyxis.NewCertificationInput(certProject)
			if err != nil {
				return fmt.Errorf(
					"%w: could not build submission with required assets: %s",
					errors.ErrSubmittingToPyxis,
					err,
				)
			}

			submission.
				// The engine writes the certified image config to disk in a Pyxis-specific format.
				WithCertImageFromFile(path.Join(artifacts.Path(), certification.DefaultCertImageFilename)).
				// Include Preflight's test results in our submission. pyxis.TestResults embeds them.
				WithPreflightResultsFromFile(path.Join(artifacts.Path(), certification.DefaultTestResultsFilename)).
				// The certification engine writes the rpmManifest for images not based on scratch.
				WithRPMManifestFromFile(path.Join(artifacts.Path(), certification.DefaultRPMManifestFilename)).
				// Include the preflight execution log file.
				WithArtifactFromFile(viper.GetString("logfile"))

			input, err := submission.Finalize()
			if err != nil {
				return fmt.Errorf(
					"%w: unable to finalize data that would be sent to pyxis %s",
					errors.ErrSubmittingToPyxis,
					err,
				)
			}

			certResults, err := pyxisClient.SubmitResults(ctx, input)
			if err != nil {
				return fmt.Errorf("%w: %s", errors.ErrSubmittingToPyxis, err)
			}

			log.Info("Test results have been submitted to Red Hat.")
			log.Info("These results will be reviewed by Red Hat for final certification.")
			log.Infof("The container's image id is: %s.", certResults.CertImage.ID)
			log.Infof("Please check %s to view scan results.", buildScanResultsURL(projectId, certResults.CertImage.ID))
			log.Infof("Please check %s to monitor the progress.", buildOverviewURL(projectId))
		}

		log.Infof("Preflight result: %s", convertPassedOverall(results.PassedOverall))

		return nil
	},
}

func pyxisHostLookup(pyxisEnv, hostOverride string) string {
	envs := map[string]string{
		"prod":  "catalog.redhat.com/api/containers",
		"uat":   "catalog.uat.redhat.com/api/containers",
		"qa":    "catalog.qa.redhat.com/api/containers",
		"stage": "catalog.stage.redhat.com/api/containers",
	}
	if hostOverride != "" {
		return hostOverride
	}

	pyxisHost, ok := envs[pyxisEnv]
	if !ok {
		pyxisHost = envs["prod"]
	}
	return pyxisHost
}

func init() {
	checkContainerCmd.Flags().BoolVarP(&submit, "submit", "s", false, "submit check container results to red hat")
	viper.BindPFlag("submit", checkContainerCmd.Flags().Lookup("submit"))

	checkContainerCmd.Flags().StringP("docker-config", "d", "", "path to docker config.json file (env: PFLT_DOCKERCONFIG)")
	viper.BindPFlag("dockerConfig", checkContainerCmd.Flags().Lookup("docker-config"))

	checkContainerCmd.Flags().String("pyxis-api-token", "", "API token for Pyxis authentication (env: PFLT_PYXIS_API_TOKEN)")
	viper.BindPFlag("pyxis_api_token", checkContainerCmd.Flags().Lookup("pyxis-api-token"))

	checkContainerCmd.Flags().String("pyxis-host", "", fmt.Sprintf("Host to use for Pyxis submissions.\nThis will override Pyxis Env.\nOnly set this if you know what you are doing.\nIf you do set it, it should include just the host, and the URI path.\n(env: PFLT_PYXIS_HOST)"))
	viper.BindPFlag("pyxis_host", checkContainerCmd.Flags().Lookup("pyxis-host"))

	checkContainerCmd.Flags().String("pyxis-env", certification.DefaultPyxisEnv, "Env to use for Pyxis submissions.")
	viper.BindPFlag("pyxis_env", checkContainerCmd.Flags().Lookup("pyxis-env"))

	checkContainerCmd.Flags().String("certification-project-id", "", fmt.Sprintf("Certification Project ID from connect.redhat.com.\nShould be supplied without the ospid- prefix.\n(env: PFLT_CERTIFICATION_PROJECT_ID)"))
	viper.BindPFlag("certification_project_id", checkContainerCmd.Flags().Lookup("certification-project-id"))

	checkCmd.AddCommand(checkContainerCmd)
}
