//nolint:unused
package lib

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/pyxis"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/runtime"
)

type (
	fibdFunc func(ctx context.Context, digests []string) ([]pyxis.CertImage, error)
	gpFunc   func(context.Context) (*pyxis.CertProject, error)
	srFunc   func(context.Context, *pyxis.CertificationInput) (*pyxis.CertificationResults, error)
)

func NewFakePyxisClientNoop() *FakePyxisClient {
	return &FakePyxisClient{
		findImagesByDigestFunc: fidbFuncNoop,
		getProjectsFunc:        gpFuncNoop,
		submitResultsFunc:      srFuncNoop,
	}
}

// FakePyxisClient is a configurable pyxisClient for use in testing. It accepts function definitions to
// use to implement a cmd.pyxisClient.
type FakePyxisClient struct {
	findImagesByDigestFunc fibdFunc
	getProjectsFunc        gpFunc
	submitResultsFunc      srFunc
}

// baseProject returns a pyxis.CertProject with an id of projectID, or a base value
// if none is provided.
func (pc *FakePyxisClient) baseProject(projectID string) pyxis.CertProject {
	pid := "000000000000"
	if len(projectID) > 0 {
		pid = projectID
	}

	return pyxis.CertProject{
		ID:                  pid,
		CertificationStatus: "false",
		Name:                "some-project",
	}
}

// successfulCertResults returns a pyxis.CertificationResults for use in tests emulating successful
// submission.
func (pc *FakePyxisClient) successfulCertResults(projectID, certImageID, testResultsID string) pyxis.CertificationResults {
	pid := "000000000000"
	if len(projectID) > 0 {
		pid = projectID
	}

	ciid := "111111111111"
	if len(certImageID) > 0 {
		ciid = certImageID
	}

	trid := "222222222222"
	if len(testResultsID) > 0 {
		trid = testResultsID
	}

	return pyxis.CertificationResults{
		CertProject: &pyxis.CertProject{
			ID: pid,
		},
		CertImage: &pyxis.CertImage{
			ID: ciid,
		},
		TestResults: &pyxis.TestResults{
			ID: trid,
		},
	}
}

// setGPFuncReturnBaseProject sets pc.getProjectFunc to a function that returns baseProject.
// This is a FakePyxisClient method because it enables standardizing on a single value of
// a CertificationProject for GetProject calls, tied to the instance of FakePyxisClient
func (pc *FakePyxisClient) setGPFuncReturnBaseProject(projectID string) {
	baseproj := pc.baseProject(projectID)
	pc.getProjectsFunc = func(context.Context) (*pyxis.CertProject, error) { return &baseproj, nil }
}

func (pc *FakePyxisClient) setSRFuncSubmitSuccessfully(projectID, certImageID string) {
	baseproj := pc.baseProject(projectID)
	certresults := pc.successfulCertResults(baseproj.ID, certImageID, "")
	pc.submitResultsFunc = func(context.Context, *pyxis.CertificationInput) (*pyxis.CertificationResults, error) {
		return &certresults, nil
	}
}

func (pc *FakePyxisClient) FindImagesByDigest(ctx context.Context, digests []string) ([]pyxis.CertImage, error) {
	return pc.findImagesByDigestFunc(ctx, digests)
}

func (pc *FakePyxisClient) GetProject(ctx context.Context) (*pyxis.CertProject, error) {
	return pc.getProjectsFunc(ctx)
}

func (pc *FakePyxisClient) SubmitResults(ctx context.Context, ci *pyxis.CertificationInput) (*pyxis.CertificationResults, error) {
	return pc.submitResultsFunc(ctx, ci)
}

// gpFuncReturnError implements gpFunc but returns an error.
func gpFuncReturnError(ctx context.Context) (*pyxis.CertProject, error) {
	return nil, errors.New("some error returned from the api")
}

// gpFuncReturnHostedRegistry implements gpFunc and returns hosted_registry=true.
func gpFuncReturnHostedRegistry(ctx context.Context) (*pyxis.CertProject, error) {
	return &pyxis.CertProject{
		ID:                  "000000000000",
		CertificationStatus: "false",
		Name:                "some-project",
		Container: pyxis.Container{
			HostedRegistry: true,
		},
	}, nil
}

// gpFuncReturnScratchException implements gpFunc and returns a scratch exception.
func gpFuncReturnScratchException(ctx context.Context) (*pyxis.CertProject, error) {
	return &pyxis.CertProject{
		Container: pyxis.Container{
			Type: "scratch",
		},
	}, nil
}

// gpFuncReturnScratchImageException implements gpFunc and returns a scratch image exception.
func gpFuncReturnScratchImageException(ctx context.Context) (*pyxis.CertProject, error) {
	return &pyxis.CertProject{
		Container: pyxis.Container{
			OsContentType: "Scratch Image",
		},
	}, nil
}

// gpFuncReturnRootException implements gpFunc and returns a root exception.
func gpFuncReturnRootException(ctx context.Context) (*pyxis.CertProject, error) {
	return &pyxis.CertProject{
		Container: pyxis.Container{
			DockerConfigJSON: "",
			Privileged:       true,
		},
	}, nil
}

// gpFuncReturnScratchRootException implements gpFunc and returns a root exception.
func gpFuncReturnScratchRootException(ctx context.Context) (*pyxis.CertProject, error) {
	return &pyxis.CertProject{
		Container: pyxis.Container{
			DockerConfigJSON: "",
			OsContentType:    "Scratch Image",
			Privileged:       true,
		},
	}, nil
}

// gpFuncReturnNoException implements gpFunc and returns no exception indicators.
func gpFuncReturnNoException(ctx context.Context) (*pyxis.CertProject, error) {
	return &pyxis.CertProject{
		Container: pyxis.Container{
			Type:       "",
			Privileged: false,
		},
	}, nil
}

// srFuncReturnError implements srFunc and returns a submission error.
func srFuncReturnError(ctx context.Context, ci *pyxis.CertificationInput) (*pyxis.CertificationResults, error) {
	return nil, errors.New("some submission error")
}

// fidbFuncNoop implements a fidbFunc, best to use while instantiating FakePyxisClient.
func fidbFuncNoop(ctx context.Context, digests []string) ([]pyxis.CertImage, error) {
	return nil, nil
}

// gpFuncNoop implements a gpFunc, best to use while instantiating FakePyxisClient.
func gpFuncNoop(ctx context.Context) (*pyxis.CertProject, error) {
	return nil, nil
}

// srFuncNoop implements a srFuncNoop, best to use while instantiating FakePyxisClient.
func srFuncNoop(ctx context.Context, ci *pyxis.CertificationInput) (*pyxis.CertificationResults, error) {
	return nil, nil
}

// fakeCheckEngine implements a certification.CheckEngine with configurables for use in tests.
type fakeCheckEngine struct {
	image              string
	passed             bool
	errorRunningChecks bool
	errorMsg           string
}

// generateCheck generates a check with a randomized name
func (e fakeCheckEngine) generateCheck() check.Check {
	generatedName := fmt.Sprintf("test-rand-%d", rand.Int())

	doNothing := func(c context.Context, i image.ImageReference) (bool, error) {
		return true, nil
	}

	return check.NewGenericCheck(generatedName,
		doNothing,
		check.Metadata{},
		check.HelpText{},
		nil,
	)
}

func (e fakeCheckEngine) ExecuteChecks(ctx context.Context) error {
	if e.errorRunningChecks {
		return errors.New(e.errorMsg)
	}
	return nil
}

func (e fakeCheckEngine) Results(ctx context.Context) certification.Results {
	return certification.Results{
		TestedImage:   "",
		PassedOverall: false,
		TestedOn: runtime.OpenshiftClusterVersion{
			Name:    "FakeName",
			Version: "FakeVersion",
		},
		CertificationHash: "",
		Passed: []certification.Result{
			{Check: e.generateCheck(), ElapsedTime: 20 * time.Millisecond},
		},
		Failed: []certification.Result{},
		Errors: []certification.Result{},
	}
}

// badResultWriter implements ResultWriter and will automatically fail with the
// provided errmsg.
type badResultWriter struct {
	errmsg string
}

func (brw *badResultWriter) OpenFile(n string) (io.WriteCloser, error) {
	return nil, errors.New(brw.errmsg)
}

func (brw *badResultWriter) Close() error {
	return nil
}

func (brw *badResultWriter) Write(p []byte) (int, error) {
	return 0, nil
}

// badFormatter implements Formatter and fails to Format with the provided errmsg.
type badFormatter struct {
	errormsg string
}

func (f *badFormatter) FileExtension() string {
	return "fake"
}

func (f *badFormatter) PrettyName() string {
	return "Fake"
}

func (f *badFormatter) Format(ctx context.Context, r certification.Results) ([]byte, error) {
	return nil, errors.New(f.errormsg)
}

// badResultSubmitter implements ResultSubmitter and fails to submit with the included errmsg.
type badResultSubmitter struct {
	errmsg string
}

func (brs *badResultSubmitter) Submit(ctx context.Context) error {
	return errors.New(brs.errmsg)
}
