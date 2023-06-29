package pyxis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// certificationInputBuilder facilitates the building of CertificationInput for
// submitting an asset to Pyxis.
type certificationInputBuilder struct {
	CertificationInput
}

// NewCertificationInput accepts required values for submitting to Pyxis, and returns a CertificationInputBuilder for
// adding additional files as artifacts to the submission. The caller must call finalize() in order to receive
// a *CertificationInput.
func NewCertificationInput(ctx context.Context, project *CertProject, opts ...CertificationInputOption) (*CertificationInput, error) {
	if project == nil {
		return nil, fmt.Errorf("a certification project was not provided and is required")
	}

	b := certificationInputBuilder{
		CertificationInput: CertificationInput{
			CertProject: project,
		},
	}

	for _, opt := range opts {
		if err := opt(&b); err != nil {
			return nil, fmt.Errorf("certification input error: %v", err)
		}
	}

	return b.finalize()
}

// finalize runs a collection of safeguards to try to ensure we get a reliable
// CertificationInput. This also wires up information that's shared across
// the various included assets (e.g. ISVPID) where applicable, and returns an
// unmodifiable CertificationInput.
//
// If any required values are not included, an error is thrown.
func (b *certificationInputBuilder) finalize() (*CertificationInput, error) {
	// safeguards, make sure things aren't nil for any reason.
	if b.CertImage == nil {
		return nil, fmt.Errorf("a CertImage was not provided and is required")
	}
	if b.TestResults == nil {
		return nil, fmt.Errorf("test results were not provided and are required")
	}

	if b.RpmManifest == nil && !b.CertProject.ScratchProject() {
		return nil, fmt.Errorf("the RPM manifest was not provided and is required")
	}

	if b.Artifacts == nil {
		// we assume artifacts can be empty, but not nil.
		b.Artifacts = []Artifact{}
	}

	// connect values from different components as necessary.
	b.CertImage.ISVPID = b.CertProject.Container.ISVPID
	b.CertImage.Certified = b.TestResults.Passed

	return &b.CertificationInput, nil
}

type CertificationInputOption func(*certificationInputBuilder) error

// WithCertImage adds a pyxis.CertImage from the passed io.Reader to the CertificationInput.
// Errors are logged, but will not halt execution.
func WithCertImage(r io.Reader) CertificationInputOption {
	return func(b *certificationInputBuilder) error {
		if err := b.storeCertImage(r); err != nil {
			return fmt.Errorf("cert image could not be stored: %v", err)
		}
		return nil
	}
}

// WithPreflightResults adds formatters.UserResponse from the passed io.Reader to the CertificationInput.
// Errors are logged, but will not halt execution.
func WithPreflightResults(r io.Reader) CertificationInputOption {
	return func(b *certificationInputBuilder) error {
		if err := b.storePreflightResults(r); err != nil {
			return fmt.Errorf("preflight results could not be stored: %v", err)
		}
		return nil
	}
}

// WithRPMManifest adds the pyxis.RPMManifest from the passed io.Reader to the CertificationInput.
// Errors are logged, but will not halt execution.
func WithRPMManifest(r io.Reader) CertificationInputOption {
	return func(b *certificationInputBuilder) error {
		if err := b.storeRPMManifest(r); err != nil {
			return fmt.Errorf("rpm manifest could not be stored: %v", err)
		}
		return nil
	}
}

// WithArtifact reads from the io.Reader and binds it as an artifact to include
// in the submission. Multiple calls to this will append artifacts. Errors are logged,
// but will not halt execution. The filename parameter will be used as the Filename
// field in the Artifact struct. It will be sent as is. It should prepresent only the
// base filename.
func WithArtifact(r io.Reader, filename string) CertificationInputOption {
	return func(b *certificationInputBuilder) error {
		bts, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("artifact could not be stored: %s: %v", filename, err)
		}

		newArtifact := Artifact{
			CertProject: b.CertProject.ID,
			Content:     base64.StdEncoding.EncodeToString(bts),
			ContentType: http.DetectContentType(bts),
			Filename:    filename,
			FileSize:    int64(len(bts)),
		}

		b.Artifacts = append(b.Artifacts, newArtifact)

		return nil
	}
}

func readAndUnmarshal(r io.Reader, submission interface{}) error {
	bytes, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, &submission)
	if err != nil {
		return fmt.Errorf(
			"data for %T appears to be malformed: %w",
			submission,
			err,
		)
	}

	return nil
}

// storeRPMManifest reads the manifest from disk at path and stores it in
// the CertificationInput as an RPMManifest struct.
func (b *certificationInputBuilder) storeRPMManifest(r io.Reader) error {
	var manifest RPMManifest
	err := readAndUnmarshal(r, &manifest)
	if err != nil {
		return err
	}

	b.RpmManifest = &manifest
	return nil
}

// storePreflightResults reads the results from disk at path and stores it in
// the CertificationInput as TestResults.
func (b *certificationInputBuilder) storePreflightResults(r io.Reader) error {
	var testResults TestResults
	err := readAndUnmarshal(r, &testResults)
	if err != nil {
		return err
	}

	b.TestResults = &testResults
	return nil
}

// storeCertImage reads the image from disk at path and stores it in
// the CertificationInput as a CertImage
func (b *certificationInputBuilder) storeCertImage(r io.Reader) error {
	var image CertImage
	err := readAndUnmarshal(r, &image)
	if err != nil {
		return err
	}

	b.CertImage = &image
	return nil
}
