package shell

import (
	"errors"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
)

func TestShell(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Shell Suite")
}

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.TraceLevel)
}

var (
	originalPodmanEngine      cli.PodmanEngine
	originalSkopeoEngine      cli.SkopeoEngine
	originalOperatorSdkEngine cli.OperatorSdkEngine
)

var _ = BeforeSuite(func() {
	originalPodmanEngine = podmanEngine
	originalSkopeoEngine = skopeoEngine
	originalOperatorSdkEngine = operatorSdkEngine
})

var _ = AfterSuite(func() {
	podmanEngine = originalPodmanEngine
	skopeoEngine = originalSkopeoEngine
	operatorSdkEngine = originalOperatorSdkEngine
})

/*
------------------- Podman Engine ---------------------
*/

// This struct is meant to implement cli.PodmanEngine
// It is used for unit testing, allowing the package-level
// variable of podmanEngine to be overridden in test files

type FakePodmanEngine struct {
	RunReportStdout     string
	RunReportStderr     string
	RunReportExitCode   int
	PullReportStdouterr string
	ImageInspectReport  cli.ImageInspectReport
	ImageScanReport     cli.ImageScanReport
}

func (fpe FakePodmanEngine) Run(opts cli.ImageRunOptions) (*cli.ImageRunReport, error) {
	runReport := cli.ImageRunReport{
		Stdout:   fpe.RunReportStdout,
		Stderr:   fpe.RunReportStderr,
		ExitCode: fpe.RunReportExitCode,
	}
	return &runReport, nil
}

func (fpe FakePodmanEngine) Pull(rawImage string, opts cli.ImagePullOptions) (*cli.ImagePullReport, error) {
	pullReport := cli.ImagePullReport{
		StdoutErr: fpe.PullReportStdouterr,
	}

	return &pullReport, nil
}

func (fpe FakePodmanEngine) Save(nameOrID string, tags []string, opts cli.ImageSaveOptions) error {
	return nil
}

func (fpe FakePodmanEngine) InspectImage(rawImage string, opts cli.ImageInspectOptions) (*cli.ImageInspectReport, error) {
	return &fpe.ImageInspectReport, nil
}

func (fpe FakePodmanEngine) ScanImage(rawImage string) (*cli.ImageScanReport, error) {
	return &fpe.ImageScanReport, nil
}

type BadPodmanEngine struct{}

func (bpe BadPodmanEngine) Run(cli.ImageRunOptions) (*cli.ImageRunReport, error) {
	runReport := cli.ImageRunReport{
		Stdout:   "Bad stadout",
		Stderr:   "Bad stderr",
		ExitCode: -1,
	}
	return &runReport, errors.New("the Podman Run has failed")
}

func (bpe BadPodmanEngine) Pull(rawImage string, opts cli.ImagePullOptions) (*cli.ImagePullReport, error) {
	pullReport := cli.ImagePullReport{
		StdoutErr: "Bad stdouterr",
	}
	return &pullReport, errors.New("the Podman Pull has failed")
}

func (bpe BadPodmanEngine) Save(nameOrID string, tags []string, opts cli.ImageSaveOptions) error {
	return errors.New("the Podman Save has failed")
}

func (bpe BadPodmanEngine) InspectImage(rawImage string, opts cli.ImageInspectOptions) (*cli.ImageInspectReport, error) {
	return nil, errors.New("the Podman Inspect Image has failed")
}

func (bpe BadPodmanEngine) ScanImage(rawImage string) (*cli.ImageScanReport, error) {
	return nil, errors.New("the Podman Scan Image has failed")
}

/*
------------------- Skopeo Engine ---------------------
*/

type FakeSkopeoEngine struct {
	SkopeoReportStdout string
	SkopeoReportStderr string
	Tags               []string
}

type SkopeoData struct {
	Repository string
	Tags       []string
}

func (fse FakeSkopeoEngine) ListTags(image string) (*cli.SkopeoListTagsReport, error) {
	skopeoReport := cli.SkopeoListTagsReport{
		Stdout: fse.SkopeoReportStdout,
		Stderr: fse.SkopeoReportStderr,
		Tags:   fse.Tags,
	}
	return &skopeoReport, nil
}

type BadSkopeoEngine struct{}

func (bse BadSkopeoEngine) ListTags(string) (*cli.SkopeoListTagsReport, error) {
	skopeoReport := cli.SkopeoListTagsReport{
		Stdout: "Bad Stdout",
		Stderr: "Bad stderr",
		Tags:   []string{""},
	}
	return &skopeoReport, errors.New("the Skopeo ListTags has failed")
}

/*
------------------- Operator Sdk Engine -------------------
*/

type FakeOperatorSdkEngine struct {
	OperatorSdkReport   cli.OperatorSdkScorecardReport
	OperatorSdkBVReport cli.OperatorSdkBundleValidateReport
}

func (fose FakeOperatorSdkEngine) Scorecard(bundleImage string, opts cli.OperatorSdkScorecardOptions) (*cli.OperatorSdkScorecardReport, error) {
	return &fose.OperatorSdkReport, nil
}

func (fose FakeOperatorSdkEngine) BundleValidate(bundleImage string, opts cli.OperatorSdkBundleValidateOptions) (*cli.OperatorSdkBundleValidateReport, error) {
	return &fose.OperatorSdkBVReport, nil
}

type BadOperatorSdkEngine struct{}

func (bose BadOperatorSdkEngine) Scorecard(bundleImage string, opts cli.OperatorSdkScorecardOptions) (*cli.OperatorSdkScorecardReport, error) {
	operatorSdkReport := cli.OperatorSdkScorecardReport{
		Stdout: "Bad Stdout",
		Stderr: "Bad Stderr",
		Items:  []cli.OperatorSdkScorecardItem{},
	}
	return &operatorSdkReport, errors.New("the Operator Sdk Scorecard has failed")
}

func (bose BadOperatorSdkEngine) BundleValidate(bundleImage string, opts cli.OperatorSdkBundleValidateOptions) (*cli.OperatorSdkBundleValidateReport, error) {
	operatorSdkReport := cli.OperatorSdkBundleValidateReport{
		Stdout:  "Bad Stdout",
		Stderr:  "Bad Stderr",
		Passed:  false,
		Outputs: []cli.OperatorSdkBundleValidateOutput{},
	}
	return &operatorSdkReport, errors.New("the Operator Sdk Bundle Validate has failed")
}
