package shell

import (
	"errors"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
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
	originalEngine cli.PodmanEngine
)

var _ = BeforeSuite(func() {
	originalEngine = podmanEngine
})

var _ = AfterSuite(func() {
	podmanEngine = originalEngine
})

// This struct is meant to implement cli.PodmanEngine
// It is used for unit testing, allowing the package-level
// variable of podmanEngine to be overridden in test files

type FakePodmanEngine struct {
	RunReportStdout     string
	RunReportStderr     string
	PullReportStdouterr string
	ImageInspectReport  cli.ImageInspectReport
}

func (fpe FakePodmanEngine) Run(opts cli.ImageRunOptions) (*cli.ImageRunReport, error) {
	runReport := cli.ImageRunReport{
		Stdout: fpe.RunReportStdout,
		Stderr: fpe.RunReportStderr,
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

type BadPodmanEngine struct{}

func (bpe BadPodmanEngine) Run(cli.ImageRunOptions) (*cli.ImageRunReport, error) {
	runReport := cli.ImageRunReport{
		Stdout: "Bad stadout",
		Stderr: "Bad stderr",
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

func checkPodmanErrors(check certification.Check) func() {
	return func() {
		Describe("Checking that Podman errors are handled correctly", func() {
			BeforeEach(func() {
				fakeEngine := BadPodmanEngine{}
				podmanEngine = fakeEngine
			})
			Context("When PodMan throws an error", func() {
				It("should fail Validate and return an error", func() {
					ok, err := check.Validate("dummy/image")
					Expect(err).To(HaveOccurred())
					Expect(ok).To(BeFalse())
				})
			})
		})

	}
}

func checkShouldPassValidate(check certification.Check) func() {
	return func() {
		ok, err := check.Validate("dummy/image")
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())
	}
}

func checkShouldNotPassValidate(check certification.Check) func() {
	return func() {
		ok, err := check.Validate("dummy/image")
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeFalse())
	}
}
