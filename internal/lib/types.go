package lib

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/spf13/viper"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/pyxis"

	log "github.com/sirupsen/logrus"
)

// ResultWriter defines methods associated with writing check results.
type ResultWriter interface {
	OpenFile(name string) (io.WriteCloser, error)
	io.WriteCloser
}

// ResultSubmitter defines methods associated with submitting results to Red HAt.
type ResultSubmitter interface {
	Submit(context.Context) error
}

// PyxisClient defines pyxis API interactions that are relevant to check executions in cmd.
type PyxisClient interface {
	FindImagesByDigest(ctx context.Context, digests []string) ([]pyxis.CertImage, error)
	GetProject(context.Context) (*pyxis.CertProject, error)
	SubmitResults(context.Context, *pyxis.CertificationInput) (*pyxis.CertificationResults, error)
}

// newPyxisClient initializes a pyxisClient with relevant information from cfg.
// If the the CertificationProjectID, PyxisAPIToken, or PyxisHost are empty, then nil is returned.
// Callers should treat a nil pyxis client as an indicator that pyxis calls should not be made.
//
//nolint:unparam // ctx is unused. Keep for future use.
func NewPyxisClient(ctx context.Context, cfg certification.Config) PyxisClient {
	if cfg.CertificationProjectID() == "" || cfg.PyxisAPIToken() == "" || cfg.PyxisHost() == "" {
		return nil
	}

	return pyxis.NewPyxisClient(
		cfg.PyxisHost(),
		cfg.PyxisAPIToken(),
		cfg.CertificationProjectID(),
		&http.Client{Timeout: 60 * time.Second},
	)
}

// ContainerCertificationSubmitter submits container results to Pyxis, and implements
// a ResultSubmitter.
type ContainerCertificationSubmitter struct {
	CertificationProjectID string
	Pyxis                  PyxisClient
	DockerConfig           string
	PreflightLogFile       string
}

func (s *ContainerCertificationSubmitter) Submit(ctx context.Context) error {
	log.Info("preparing results that will be submitted to Red Hat")

	// get the project info from pyxis
	certProject, err := s.Pyxis.GetProject(ctx)
	if err != nil {
		return fmt.Errorf("could not retrieve project: %w", err)
	}

	// Ensure that a certProject was returned. In theory we would expect pyxis
	// to throw an error if no project is returned, but in the event that it doesn't
	// we need to confirm before we proceed in order to prevent a runtime panic
	// setting the DockerConfigJSON below.
	if certProject == nil {
		return fmt.Errorf("no certification project was returned from pyxis")
	}

	log.Tracef("CertProject: %+v", certProject)

	// only read the dockerfile if the user provides a location for the file
	// at this point in the flow, if `cfg.DockerConfig` is empty we know the repo is public and can continue the submission flow
	if s.DockerConfig != "" {
		dockerConfigJSONBytes, err := os.ReadFile(s.DockerConfig)
		if err != nil {
			return fmt.Errorf("could not open file for submission: %s: %w",
				s.DockerConfig,
				err,
			)
		}

		certProject.Container.DockerConfigJSON = string(dockerConfigJSONBytes)
	}

	// the below code is for the edge case where a partner has a DockerConfig in pyxis, but does not send one to preflight.
	// when we call pyxis's GetProject API, we get back the DockerConfig as a PGP encrypted string and not JSON,
	// if we were to send what pyixs just sent us in a update call, pyxis would throw a validation error saying it's not valid json
	// the below code aims to set the DockerConfigJSON to an empty string, and since this field is `omitempty` when we marshall it
	// we will not get a validation error
	if s.DockerConfig == "" {
		certProject.Container.DockerConfigJSON = ""
	}

	// prepare submission. We ignore the error because nil checks for the certProject
	// are done earlier to prevent panics, and that's the only error case for this function.
	submission, _ := pyxis.NewCertificationInput(certProject)

	certImage, err := os.Open(path.Join(artifacts.Path(), certification.DefaultCertImageFilename))
	if err != nil {
		return fmt.Errorf("could not open file for submission: %s: %w",
			certification.DefaultCertImageFilename,
			err,
		)
	}
	defer certImage.Close()

	preflightResults, err := os.Open(path.Join(artifacts.Path(), certification.DefaultTestResultsFilename))
	if err != nil {
		return fmt.Errorf(
			"could not open file for submission: %s: %w",
			certification.DefaultTestResultsFilename,
			err,
		)
	}
	defer preflightResults.Close()

	rpmManifest, err := os.Open(path.Join(artifacts.Path(), certification.DefaultRPMManifestFilename))
	if err != nil {
		return fmt.Errorf(
			"could not open file for submission: %s: %w",
			certification.DefaultRPMManifestFilename,
			err,
		)
	}
	defer rpmManifest.Close()

	logfile, err := os.Open(s.PreflightLogFile)
	if err != nil {
		return fmt.Errorf(
			"could not open file for submission: %s: %w",
			s.PreflightLogFile,
			err,
		)
	}
	defer logfile.Close()

	submission.
		// The engine writes the certified image config to disk in a Pyxis-specific format.
		WithCertImage(certImage).
		// Include Preflight's test results in our submission. pyxis.TestResults embeds them.
		WithPreflightResults(preflightResults).
		// The certification engine writes the rpmManifest for images not based on scratch.
		WithRPMManifest(rpmManifest).
		// Include the preflight execution log file.
		WithArtifact(logfile, filepath.Base(s.PreflightLogFile))

	input, err := submission.Finalize()
	if err != nil {
		return fmt.Errorf("unable to finalize data that would be sent to pyxis: %w", err)
	}

	certResults, err := s.Pyxis.SubmitResults(ctx, input)
	if err != nil {
		return fmt.Errorf("could not submit to pyxis: %w", err)
	}

	log.Info("Test results have been submitted to Red Hat.")
	log.Info("These results will be reviewed by Red Hat for final certification.")
	log.Infof("The container's image id is: %s.", certResults.CertImage.ID)
	log.Infof("Please check %s to view scan results.", BuildScanResultsURL(s.CertificationProjectID, certResults.CertImage.ID))
	log.Infof("Please check %s to monitor the progress.", BuildOverviewURL(s.CertificationProjectID))

	return nil
}

// NoopSubmitter is a no-op ResultSubmitter that optionally logs a message
// and a reason as to why results were not submitted.
type NoopSubmitter struct {
	emitLog bool
	reason  string
	log     *log.Logger
}

func NewNoopSubmitter(emitLog bool, log *log.Logger) *NoopSubmitter {
	return &NoopSubmitter{
		emitLog: emitLog,
		log:     log,
	}
}

var _ ResultSubmitter = &NoopSubmitter{}

func (s *NoopSubmitter) Submit(ctx context.Context) error {
	if s.emitLog {
		msg := "Results are not being sent for submission."
		if s.reason != "" {
			msg = fmt.Sprintf("%s Reason: %s.", msg, s.reason)
		}

		s.log.Info(msg)
	}

	return nil
}

func (s *NoopSubmitter) SetEmitLog(emitLog bool) {
	s.emitLog = emitLog
}

func (s *NoopSubmitter) SetReason(reason string) {
	s.reason = reason
}

func BuildConnectURL(projectID string) string {
	connectURL := fmt.Sprintf("https://connect.redhat.com/projects/%s", projectID)

	pyxisEnv := viper.GetString("pyxis_env")
	if len(pyxisEnv) > 0 && pyxisEnv != "prod" {
		connectURL = fmt.Sprintf("https://connect.%s.redhat.com/projects/%s", viper.GetString("pyxis_env"), projectID)
	}

	return connectURL
}

func BuildOverviewURL(projectID string) string {
	return fmt.Sprintf("%s/overview", BuildConnectURL(projectID))
}

func BuildScanResultsURL(projectID string, imageID string) string {
	return fmt.Sprintf("%s/images/%s/scan-results", BuildConnectURL(projectID), imageID)
}
