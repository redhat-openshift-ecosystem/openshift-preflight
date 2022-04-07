package cmd

import (
	"context"
	"encoding/base64"
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
			// Since we want the modified version, write it back
			// to viper so that subsequent calls don't need to check
			viper.Set("certification_project_id", projectId)
		}
		apiToken := viper.GetString("pyxis_api_token")
		pyxisClient := pyxis.NewPyxisClient(viper.GetString("pyxis_host"), apiToken, projectId, &http.Client{Timeout: 60 * time.Second})
		certProject, err := pyxisClient.GetProject(ctx)
		if err != nil {
			log.Error(err, "could not retrieve project")
			return err
		}
		log.Debugf("Certification project name is: %s", certProject.Name)
		if certProject.Container.OsContentType == "scratch" {
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
			log.Tracef("CertProject: %+v", certProject) // TODO() This depends on Pyxis calls made outside of this block.

			// The engine writes the certified image config to disk in a Pyxis-specific format.
			// Include that data in our submission.
			certImageFilepath := path.Join(artifacts.Path(), certification.DefaultCertImageFilename)
			log.Tracef("Loading certImage data from path %s", certImageFilepath)
			certImageBytes, err := os.ReadFile(certImageFilepath)
			if err != nil {
				return fmt.Errorf(
					"%w: unable to read file from disk to include in submission: %s: %s",
					errors.ErrSubmittingToPyxis,
					certImageFilepath,
					err,
				)
			}

			var certImage pyxis.CertImage
			err = json.Unmarshal(certImageBytes, &certImage)
			if err != nil {
				return fmt.Errorf(
					"%w: data for the %s appears to be malformed: %s",
					errors.ErrSubmittingToPyxis,
					"certImage",
					err,
				)
			}

			// The certification engine also writes the rpmManifest for images not based on scratch.
			// Include that data in our submission.
			rpmManifestFilepath := path.Join(artifacts.Path(), certification.DefaultRPMManifestFilename)
			log.Tracef("Loading RPM manifest data from path %s", rpmManifestFilepath)
			rpmManifestBytes, err := os.ReadFile(rpmManifestFilepath)
			if err != nil {
				return fmt.Errorf(
					"%w: unable to read file from disk to include in submission: %s: %s",
					errors.ErrSubmittingToPyxis,
					rpmManifestFilepath,
					err,
				)
			}

			var rpmManifest pyxis.RPMManifest
			err = json.Unmarshal(rpmManifestBytes, &rpmManifest)
			if err != nil {
				return fmt.Errorf(
					"%w: data for the %s appears to be malformed: %s",
					errors.ErrSubmittingToPyxis,
					"rpm manifest",
					err,
				)
			}

			// Include Preflight's test results in our submission. pyxis.TestResults embeds them.
			var testResults pyxis.TestResults
			err = json.Unmarshal(formattedResults, &testResults)
			if err != nil {
				return fmt.Errorf(
					"%w: data for the %s appears to be malformed: %s",
					errors.ErrSubmittingToPyxis,
					"preflight results.json",
					err,
				)
			}

			// Populate the certImage with missing values as a result of this execution.
			certImage.ISVPID = certProject.Container.ISVPID // NOTE: certProject depends on Pyxis calls made outside this block.
			certImage.Certified = testResults.Passed

			// Send the preflight logfile as an artifact in our submission.
			logFileName := viper.GetString("logfile")
			logFileBytes, logFileSize, err := readFileAndGetSize(logFileName)
			if err != nil {
				return fmt.Errorf(
					"%w: unable to read file from disk to include in submission %s: %s",
					errors.ErrSubmittingToPyxis,
					logFileName,
					err,
				)
			}

			logFileArtifact := pyxis.Artifact{
				CertProject: projectId,
				Content:     base64.StdEncoding.EncodeToString(logFileBytes),
				ContentType: http.DetectContentType(logFileBytes),
				Filename:    logFileName,
				FileSize:    logFileSize,
			}

			// additional artifacts to send to Pyxis.
			artifacts := []pyxis.Artifact{logFileArtifact}

			// assemble all of the components into the expected POST input format.
			// and submit the results.
			certInput := &pyxis.CertificationInput{
				CertProject: certProject,
				CertImage:   &certImage,
				RpmManifest: &rpmManifest,
				TestResults: &testResults,
				Artifacts:   artifacts,
			}

			certResults, err := pyxisClient.SubmitResults(ctx, certInput) // TODO() This pyxisClient is established outside of this block.
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

func init() {
	checkContainerCmd.Flags().BoolVarP(&submit, "submit", "s", false, "submit check container results to red hat")
	viper.BindPFlag("submit", checkContainerCmd.Flags().Lookup("submit"))

	checkContainerCmd.Flags().StringP("docker-config", "d", "", "path to docker config.json file")
	viper.BindPFlag("dockerConfig", checkContainerCmd.Flags().Lookup("docker-config"))

	checkContainerCmd.Flags().String("pyxis-api-token", "", "API token for Pyxis authentication")
	checkContainerCmd.MarkFlagRequired("pyxis-api-token")
	viper.BindPFlag("pyxis_api_token", checkContainerCmd.Flags().Lookup("pyxis-api-token"))

	checkContainerCmd.Flags().String("pyxis-host", certification.DefaultPyxisHost, "Host to use for Pyxis submissions.")
	viper.BindPFlag("pyxis_host", checkContainerCmd.Flags().Lookup("pyxis-host"))

	checkContainerCmd.Flags().String("certification-project-id", "", "Certification Project ID from connect.redhat.com. Should be supplied without the ospid- prefix.")
	checkContainerCmd.MarkFlagRequired("certification-project-id")
	viper.BindPFlag("certification_project_id", checkContainerCmd.Flags().Lookup("certification-project-id"))

	checkCmd.AddCommand(checkContainerCmd)
}
