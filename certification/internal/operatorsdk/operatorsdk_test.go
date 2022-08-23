package operatorsdk

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	testStdoutValue               = `{}`
	testBundleValidateStdoutValue = `{"passed": true, "outputs": null}`
)

var _ = Describe("OperatorSdk", func() {
	var tmpdir string
	BeforeEach(func() {
		var err error
		tmpdir, err = os.MkdirTemp("", "operatorsdk-test-artifacts-*")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(os.RemoveAll, tmpdir)
		artifacts.SetDir(tmpdir)
	})
	When("The Scorecard result is good", func() {
		It("should succeed", func() {
			operatorSdk := New("foo.image", fakeExecCommandSuccess)
			_, err := operatorSdk.Scorecard(context.TODO(), "foo.image", OperatorSdkScorecardOptions{
				ResultFile:     "success.txt",
				OutputFormat:   "json",
				Selector:       []string{"selector1", "selector2"},
				Kubeconfig:     "/my/kubeconfig",
				Namespace:      "awesome-namespace",
				ServiceAccount: "this-service-account",
				Verbose:        true,
				WaitTime:       "120m",
			})
			Expect(err).ToNot(HaveOccurred())
		})
	})
	When("The Scorecard result is a failure", func() {
		It("should fail", func() {
			operatorSdk := New("foo.image", fakeExecCommandFailure)
			_, err := operatorSdk.Scorecard(context.TODO(), "foo.image", OperatorSdkScorecardOptions{
				ResultFile:   "failure.txt",
				OutputFormat: "text",
			})
			Expect(err).To(HaveOccurred())
		})
	})
	When("The Bundle Validate result is good", func() {
		It("should succeed", func() {
			operatorSdk := New("foo.image", fakeExecValidateCommandSuccess)
			_, err := operatorSdk.BundleValidate(context.TODO(), "foo.image", OperatorSdkBundleValidateOptions{
				OutputFormat:    "json",
				LogLevel:        "debug",
				ContainerEngine: "podman",
				Selector:        []string{"selector1", "selector2"},
				OptionalValues:  map[string]string{"foo": "bar"},
				Verbose:         true,
				WaitTime:        "120m",
			})
			Expect(err).ToNot(HaveOccurred())
		})
	})
	When("The scorecard result has an error", func() {
		It("should return result == failed", func() {
			operatoSdk := New("foo.image", fakeExecValidateCommandError)
			result, err := operatoSdk.BundleValidate(context.TODO(), "foo.image", OperatorSdkBundleValidateOptions{
				OutputFormat: "text",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Passed).To(BeFalse())
		})
	})
	When("The Bundle Validate result is bad", func() {
		It("should fail", func() {
			operatorSdk := New("foo.image", fakeExecValidateCommandFailure)
			_, err := operatorSdk.BundleValidate(context.TODO(), "foo.image", OperatorSdkBundleValidateOptions{})
			Expect(err).To(HaveOccurred())
		})
	})
})

// These will be called when the inception occurs.
// If the GO_TEST_PROCESS envvar is not "1", which would
// be the case on the full testing run, it just returns.
// If it is set, then that means we are inside the
// exec call, and can therefore print whatever we want
// to stdout, stderr, and set the return value appropriately.
// When it exits, it goes back to the original test exec.
func TestShellProcessSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, testStdoutValue)
	os.Exit(0)
}

func TestShellProcessFail(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprint(os.Stderr, "FATA")
	os.Exit(1)
}

func TestBundleValidateProcessSuccess(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, testBundleValidateStdoutValue)
	os.Exit(0)
}

func TestBundleValidateProcessError(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, "FATA")
	os.Exit(0)
}

func TestBundleValidateProcessFail(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stderr, "FATA")
	os.Exit(1)
}

// What's happening here?
//
// These are the cmdContexts that are being subbed in instead of exec.Command
// So, when the SUT calls cmdContext(...) it will use this instead.
// It replaces the command that is passed in with the test args, plus the rest
// of the original command. It then execs the test binary with these args.
// The -test.run arg is will exec JUST that function from above.
func fakeExecCommandSuccess(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestShellProcessSuccess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func fakeExecCommandFailure(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestShellProcessFail", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func fakeExecValidateCommandSuccess(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestBundleValidateProcessSuccess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func fakeExecValidateCommandError(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestBundleValidateProcessError", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func fakeExecValidateCommandFailure(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestBundleValidateProcessFail", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}
