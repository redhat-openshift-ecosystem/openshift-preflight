package pyxis

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"

	log "github.com/sirupsen/logrus"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
)

// certificationInputBuilder facilitates the building of CertificationInput for
// submitting an asset to Pyxis.
type certificationInputBuilder struct {
	certificationInput
}

// NewCertificationInput accepts required values for submitting to Pyxis, and returns a CertificationInputBuilder for
// adding additional files as artifacts to the submission. The caller must call Finalize() in order to receive
// a *CertificationInput.
func NewCertificationInput(project *CertProject) (*certificationInputBuilder, error) {
	if project == nil {
		return nil, fmt.Errorf("a certification project was not provided and is required")
	}

	b := certificationInputBuilder{
		certificationInput: certificationInput{
			CertProject: project,
		},
	}

	return &b, nil
}

// Finalize runs a collection of safeguards to try to ensure we get a reliable
// CertificationInput. This also wires up information that's shared across
// the various included assets (e.g. ISVPID) where applicable, and returns an
// unmodifiable CertificationInput.
//
// If any required values are not included, an error is thrown.
func (b *certificationInputBuilder) Finalize() (*certificationInput, error) {
	// safeguards, make sure things aren't nil for any reason.
	if b.CertImage == nil {
		return nil, fmt.Errorf("a CertImage was not provided and is required")
	}
	if b.TestResults == nil {
		return nil, fmt.Errorf("test results were not provided and are required")
	}

	if b.RpmManifest == nil {
		return nil, fmt.Errorf("the RPM manifest was not provided and is required")
	}

	if b.Artifacts == nil {
		// we assume artifacts can be empty, but not nil.
		b.Artifacts = []Artifact{}
	}

	// connect values from different components as necessary.
	b.CertImage.ISVPID = b.CertProject.Container.ISVPID
	b.CertImage.Certified = b.TestResults.Passed

	return &b.certificationInput, nil
}

// WithCertImageFromFile adds a pyxis.CertImage from a file on disk to the CertificationInput.
// Errors are logged, but will not halt execution.
func (b *certificationInputBuilder) WithCertImageFromFile(filepath string) *certificationInputBuilder {
	if err := b.storeCertImage(filepath); err != nil {
		log.Error(err)
	}

	return b
}

// WithPreflightResultsFromFile adds formatters.UserResponse from a file on disk to the CertificationInput.
// Errors are logged, but will not halt execution.
func (b *certificationInputBuilder) WithPreflightResultsFromFile(filepath string) *certificationInputBuilder {
	if err := b.storePreflightResults(filepath); err != nil {
		log.Error(err)
	}

	return b
}

// WithPreflightResultsFromFile adds the pyxis.RPMManifest from a file on disk to the CertificationInput.
// Errors are logged, but will not halt execution.
func (b *certificationInputBuilder) WithRPMManifestFromFile(filepath string) *certificationInputBuilder {
	if err := b.storeRPMManifest(filepath); err != nil {
		log.Error(err)
	}

	return b
}

// WithArtifactFromFile reads a file at path and binds it as an artifact to include
// in the submission. Multiple calls to this will append artifacts. Errors are logged,
// but will not halt execution.
func (b *certificationInputBuilder) WithArtifactFromFile(filepath string) *certificationInputBuilder {
	bytes, err := os.ReadFile(filepath)
	if err != nil {
		log.Error(err)
		return b
	}

	newArtifact := Artifact{
		CertProject: b.CertProject.ID,
		Content:     base64.StdEncoding.EncodeToString(bytes),
		ContentType: http.DetectContentType(bytes),
		Filename:    path.Base(filepath),
		FileSize:    int64(len(bytes)),
	}

	b.Artifacts = append(b.Artifacts, newArtifact)

	return b
}

func readAndUnmarshal(filepath string, submission interface{}) error {
	bytes, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf(
			"%w: unable to read file from disk to include in submission: %s: %s",
			errors.ErrSubmittingToPyxis,
			filepath,
			err,
		)
	}

	err = json.Unmarshal(bytes, &submission)
	if err != nil {
		return fmt.Errorf(
			"%w: data for %T appears to be malformed: %s",
			errors.ErrSubmittingToPyxis,
			submission,
			err,
		)
	}

	return nil
}

// storeRPMManifest reads the manifest from disk at path and stores it in
// the CertificationInput as an RPMManifest struct.
func (b *certificationInputBuilder) storeRPMManifest(filepath string) error {
	var manifest RPMManifest
	err := readAndUnmarshal(filepath, &manifest)
	if err != nil {
		return err
	}

	b.RpmManifest = &manifest
	return nil
}

// storePreflightResults reads the results from disk at path and stores it in
// the CertificationInput as TestResults.
func (b *certificationInputBuilder) storePreflightResults(filepath string) error {
	var testResults TestResults
	err := readAndUnmarshal(filepath, &testResults)
	if err != nil {
		return err
	}

	b.TestResults = &testResults
	return nil
}

// storeCertImage reads the image from disk at path and stores it in
// the CertificationInput as a CertImage
func (b *certificationInputBuilder) storeCertImage(filepath string) error {
	var image CertImage
	err := readAndUnmarshal(filepath, &image)
	if err != nil {
		return err
	}

	b.CertImage = &image
	return nil
}
