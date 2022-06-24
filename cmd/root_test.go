package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// executeCommand is used for cobra command testing. It is effectively what's seen here:
// https://github.com/spf13/cobra/blob/master/command_test.go#L34-L43. It should only
// be used in tests. Typically, you should pass rootCmd as the param for root, and your
// subcommand's invocation within args.
func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	err = root.Execute()

	return buf.String(), err
}

// executeCommandWithInput runs executeCommand but passes input to stdin. This runs executeCommand
// asyncronously so that input can be passed once the command has started running. It is recommended
// that this is executed with a timeout wrapper.
func executeCommandWithInput(root *cobra.Command, input []byte, args ...string) (cmdout string, cmderr error) {
	inBuffer := promptBuffer{bytes.NewBuffer([]byte{})}
	root.SetIn(inBuffer)

	var out string
	var err error
	var wg sync.WaitGroup

	// run executeCommand with a timeout so we're not waiting forever.
	wg.Add(1)
	go func() {
		out, err = executeCommand(root, args...)
		wg.Done()
	}()

	inBuffer.Write(input)

	// wait for the prompt to complete.
	wg.Wait()

	// This is execute error (if expected)
	return out, err
}

// runTestFuncWithTimeout runs function with timeout. Returns true if the execution
// timedout and false if it did not.
func runTestFuncWithTimeout(function func(), timeout time.Duration) (timedout bool) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan int, 1)
	go func() {
		function()
		done <- 0
	}()

	for {
		select {
		case <-done:
			return false
		case <-ctx.Done():
			return true
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

var _ = Describe("cmd package utility functions", func() {
	DescribeTable("Determine filename to which to write test results",
		func(extension, expected string) {
			// Ensure resultsFilenameWithExtension accurately joins the
			// expected default filename of "results" with the extension
			// that is provided.
			actual := resultsFilenameWithExtension(extension)
			Expect(expected).To(Equal(actual))
		},
		Entry("with an extension of txt", "txt", "results.txt"),
		Entry("with an extension of json", "json", "results.json"),
	)

	Describe("Initialize Viper configuration", func() {
		Context("when initConfig() is called", func() {
			Context("and no envvars are set", func() {
				It("should have defaults set correctly", func() {
					initConfig()
					Expect(viper.GetString("namespace")).To(Equal(DefaultNamespace))
					Expect(viper.GetString("artifacts")).To(Equal(artifacts.DefaultArtifactsDir))
					Expect(viper.GetString("logfile")).To(Equal(DefaultLogFile))
					Expect(viper.GetString("loglevel")).To(Equal(DefaultLogLevel))
				})
			})
			Context("and envvars are set", func() {
				BeforeEach(func() {
					os.Setenv("PFLT_LOGFILE", "/tmp/foo.log")
					os.Setenv("PFLT_LOGLEVEL", "trace")
				})
				It("should have overrides in place", func() {
					initConfig()
					Expect(viper.GetString("namespace")).To(Equal(DefaultNamespace))
					Expect(viper.GetString("artifacts")).To(Equal(artifacts.DefaultArtifactsDir))
					Expect(viper.GetString("logfile")).To(Equal("/tmp/foo.log"))
					Expect(viper.GetString("loglevel")).To(Equal("trace"))
				})
				AfterEach(func() {
					os.Unsetenv("PFLT_LOGFILE")
					os.Unsetenv("PFLT_LOGLEVEL")
				})
			})
		})
	})

	Describe("Pre-run configuration", func() {
		var cmd *cobra.Command
		BeforeEach(func() {
			cmd = &cobra.Command{
				PersistentPreRun: preRunConfig,
				Run:              func(cmd *cobra.Command, args []string) {},
			}
			cobra.OnInitialize(initConfig)
		})
		Context("confiuring a Cobra Command", func() {
			var tmpDir string
			BeforeEach(func() {
				var err error
				tmpDir, err = os.MkdirTemp("", "prerun-config-*")
				Expect(err).ToNot(HaveOccurred())
			})
			It("should create the logfile", func() {
				viper.Set("logfile", filepath.Join(tmpDir, "foo.log"))
				Expect(cmd.ExecuteContext(context.TODO())).To(Succeed())
				_, err := os.Stat(filepath.Join(tmpDir, "foo.log"))
				Expect(err).ToNot(HaveOccurred())
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
		})
	})
})
