package cmd

import (
	"context"
	"encoding/json"
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

		if submit && !viper.IsSet("dockerConfig") {
			cmd.MarkFlagRequired("docker-config")
		}

		return nil
	},
	// this fmt.Sprintf is in place to keep spacing consistent with cobras two spaces that's used in: Usage, Flags, etc
	Example: fmt.Sprintf("  %s", "preflight check container quay.io/repo-name/container-name:version"),
	PreRun:  preRunConfig,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Expect exactly one positional arg. Check here instead of using builtin Args key
		// so that we can get a more user-friendly error message

		log.Info("certification library version ", version.Version.String())

		ctx := context.Background()

		containerImage := args[0]

		cfg := runtime.Config{
			Image:          containerImage,
			EnabledChecks:  engine.ContainerPolicy(),
			ResponseFormat: DefaultOutputFormat,
		}

		projectId := viper.GetString("certification_project_id")
		if projectId == "" {
			return errors.ErrEmptyProjectID
		}
		if strings.HasPrefix(projectId, "ospid-") {
			projectId = strings.Split(projectId, "-")[1]
		}
		apiToken := viper.GetString("pyxis_api_token")
		pyxisEngine := pyxis.NewPyxisEngine(apiToken, projectId, &http.Client{Timeout: 60 * time.Second})
		certProject, err := pyxisEngine.GetProject(ctx)
		if err != nil {
			log.Error(err, "could not retrieve project")
			return err
		}
		log.Debugf("Certification project name is: %s", certProject.Name)
		if certProject.OsContentType == "scratch" {
			cfg.EnabledChecks = engine.ScratchContainerPolicy()
			cfg.Scratch = true
		}

		if submit {
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
		if err := engine.ExecuteChecks(); err != nil {
			return err
		}
		results := engine.Results()

		// return results to the user and then close output files
		formattedResults, err := formatter.Format(results)
		if err != nil {
			return err
		}

		fmt.Fprintln(resultsOutputTarget, string(formattedResults))
		if err := resultsFile.Close(); err != nil {
			return err
		}

		if err := writeJunitIfEnabled(results); err != nil {
			return err
		}

		// submitting results to pxysis if submit flag is set
		if submit {
			log.Info("preparing results that will be submitted to Red Hat")
			log.Tracef("CertProject: %+v", certProject)

			testResultsJsonFile, err := os.Open(path.Join(artifacts.Path(), certification.DefaultTestResultsFilename))
			if err != nil {
				return err
			}
			defer testResultsJsonFile.Close()

			testResultsBytes, err := io.ReadAll(testResultsJsonFile)
			if err != nil {
				return err
			}

			testResults := new(pyxis.TestResults)
			err = json.Unmarshal(testResultsBytes, &testResults)
			if err != nil {
				return err
			}

			certImageJsonFile, err := os.Open(path.Join(artifacts.Path(), certification.DefaultCertImageFilename))
			if err != nil {
				return err
			}
			defer certImageJsonFile.Close()

			certImageBytes, err := io.ReadAll(certImageJsonFile)
			if err != nil {
				return err
			}

			certImage := new(pyxis.CertImage)
			err = json.Unmarshal(certImageBytes, &certImage)
			if err != nil {
				return err
			}

			certImage.ISVPID = certProject.Container.ISVPID
			certImage.Certified = testResults.Passed

			rpmManifestJsonFile, err := os.Open(path.Join(artifacts.Path(), certification.DefaultRPMManifestFilename))
			if err != nil {
				return err
			}
			defer rpmManifestJsonFile.Close()

			rpmManifestBytes, err := io.ReadAll(rpmManifestJsonFile)
			if err != nil {
				return err
			}

			rpmManifest := new(pyxis.RPMManifest)
			err = json.Unmarshal(rpmManifestBytes, rpmManifest)
			if err != nil {
				return err
			}

			_, certImage, _, err = pyxisEngine.SubmitResults(ctx, certProject, certImage, rpmManifest, testResults)
			if err != nil {
				return err
			}

			log.Info("Test results have been submitted to Red Hat.")
			log.Info("These results will be reviewed by Red Hat for final certification.")
			log.Infof("The container's image id is: %s.", certImage.ID)
			log.Infof("Please check %s to view scan results.", buildScanResultsURL(projectId, certImage.ID))
			log.Infof(fmt.Sprintf("Please check %s to monitor the progress.", buildOverviewURL(projectId)))
		}

		return nil
	},
}

func init() {
	checkContainerCmd.Flags().BoolVarP(&submit, "submit", "s", false, "submit check container results to red hat")
	viper.BindPFlag("submit", checkContainerCmd.Flags().Lookup("submit"))

	checkContainerCmd.Flags().StringP("docker-config", "d", "", "path to docker config.json file")
	viper.BindPFlag("dockerConfig", checkContainerCmd.Flags().Lookup("docker-config"))

	checkContainerCmd.Flags().String("pyxis-api-token", "", "API token for Pyxis authentication")
	checkContainerCmd.MarkFlagRequired("pyxis-api-token")
	viper.BindPFlag("pyxis_api_token", checkContainerCmd.Flags().Lookup("pyxis-api-token"))

	checkContainerCmd.Flags().String("pyxis-host", DefaultPyxisHost, "Host to use for Pyxis submissions.")
	viper.BindPFlag("pyxis_host", checkContainerCmd.Flags().Lookup("pyxis-host"))

	checkContainerCmd.Flags().String("certification-project-id", "", "Certification Project ID from conenct.redhat.com. Should be supplied without the ospid- prefix.")
	checkContainerCmd.MarkFlagRequired("certification-project-id")
	viper.BindPFlag("certification_project_id", checkContainerCmd.Flags().Lookup("certification-project-id"))

	checkCmd.AddCommand(checkContainerCmd)
}

func buildOverviewURL(projectID string) string {
	connectURL := fmt.Sprintf("https://connect.redhat.com/projects/%s/overview", projectID)
	pyxisHost := viper.GetString("pyxis_host")
	s := strings.Split(pyxisHost, ".")

	if pyxisHost != DefaultPyxisHost && len(s) > 3 {
		env := s[1]
		connectURL = fmt.Sprintf("https://connect.%s.redhat.com/projects/%s/overview", env, projectID)
	}

	return connectURL
}

func buildScanResultsURL(projectID string, imageID string) string {
	connectURL := fmt.Sprintf("https://connect.redhat.com/projects/%s/images/%s/scan-results", projectID, imageID)
	pyxisHost := viper.GetString("pyxis_host")
	s := strings.Split(pyxisHost, ".")

	if pyxisHost != DefaultPyxisHost && len(s) > 3 {
		env := s[1]
		connectURL = fmt.Sprintf("https://connect.%s.redhat.com/projects/%s/images/%s/scan-results", env, projectID, imageID)
	}

	return connectURL
}
