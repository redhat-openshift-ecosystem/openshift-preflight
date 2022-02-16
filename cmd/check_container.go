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

var checkContainerCmd = &cobra.Command{
	Use:   "container",
	Short: "Run checks for a container",
	Long:  `This command will run the Certification checks for a container image. `,
	Args: func(cmd *cobra.Command, args []string) error {
		if l, _ := cmd.Flags().GetBool("list-checks"); l {
			fmt.Println(fmt.Sprintf("\n%s\n%s%s", "The checks that will be executed are the following:", "- ",
				strings.Join(engine.ContainerPolicy(), "\n- ")))

			// exiting gracefully instead of retuning, otherwise cobra calls RunE
			os.Exit(0)
		}

		if len(args) != 1 {
			return fmt.Errorf("%w: A container image positional argument is required", errors.ErrInsufficientPosArguments)
		}

		if s, _ := cmd.Flags().GetBool("submit"); s {
			if d, _ := cmd.Flags().GetString("docker-config"); len(d) == 0 {
				return fmt.Errorf("%w: A docker configuration file must be present when calling --submit", errors.ErrNoDockerConfig)
			}
			if p, _ := cmd.Flags().GetString("pyxis-api-token"); len(p) == 0 {
				return fmt.Errorf("%w: A Pyxis API token must be present when calling --submit", errors.ErrNoPyxisAPIKey)
			}
			if p, _ := cmd.Flags().GetString("certification-project-id"); len(p) == 0 {
				return fmt.Errorf("%w: A Certification Project ID must be present when calling --submit", errors.ErrEmptyProjectID)
			}
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
		ctx := context.Background()
		pyxisEngine := pyxis.NewPyxisEngine(apiToken, projectId, &http.Client{Timeout: 60 * time.Second})
		certProject, err := pyxisEngine.GetProject(ctx)
		if err != nil {
			log.Error(err, "could not retrieve project")
			return err
		}
		log.Debugf("Certification project name is: %s", certProject.Name)
		if certProject.OsContentType == "scratch" {
			cfg.EnabledChecks = engine.ScratchContainerPolicy()
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
			0600,
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
		if s, _ := cmd.Flags().GetBool("submit"); s {
			log.Info("preparing results that will be submitted to Red Hat")

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

			testResultsJsonFile, err := os.Open(path.Join(artifacts.Path(), certification.DefaultTestResultsFilename))
			if err != nil {
				return err
			}
			defer testResultsJsonFile.Close()

			testResultsBytes, err := io.ReadAll(testResultsJsonFile)
			if err != nil {
				return err
			}

			var testResults = new(pyxis.TestResults)
			err = json.Unmarshal(testResultsBytes, &testResults)
			if err != nil {
				return err
			}

			//TODO: use the return values once we know what we need to display to the user
			_, _, _, err = pyxisEngine.SubmitResults(certProject, certImage, rpmManifest, testResults)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	checkContainerCmd.Flags().BoolP("submit", "s", false, "submit check container results to red hat")
	viper.BindPFlag("submit", checkContainerCmd.Flags().Lookup("submit"))

	checkContainerCmd.Flags().StringP("docker-config", "d", "", "path to docker config.json file")
	viper.BindPFlag("docker_config", checkContainerCmd.Flags().Lookup("docker-config"))

	checkContainerCmd.Flags().String("pyxis-api-token", "", "API token for Pyxis authentication")
	viper.BindPFlag("pyxis_api_token", checkContainerCmd.Flags().Lookup("pyxis-api-token"))

	checkContainerCmd.Flags().String("pyxis-host", DefaultPyxisHost, "Host to use for Pyxis submissions.")
	viper.BindPFlag("pyxis_host", checkContainerCmd.Flags().Lookup("pyxis-host"))

	checkContainerCmd.Flags().String("certification-project-id", "", "Certification Project ID from conenct.redhat.com. Should be supplied without the ospid- prefix.")
	viper.BindPFlag("certification_project_id", checkContainerCmd.Flags().Lookup("certification-project-id"))

	checkCmd.AddCommand(checkContainerCmd)
}
