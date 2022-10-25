package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

var _ = Describe("cmd package utility functions", func() {
	Describe("Get the root command", func() {
		Context("when calling the root command function", func() {
			It("should return a root command", func() {
				cmd := rootCmd()
				Expect(cmd).ToNot(BeNil())
				Expect(cmd.Commands()).ToNot(BeEmpty())
			})
		})
	})

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
		})
		Context("configuring a Cobra Command", func() {
			var tmpDir string
			BeforeEach(func() {
				var err error
				tmpDir, err = os.MkdirTemp("", "prerun-config-*")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(os.RemoveAll, tmpDir)
			})
			It("should create the logfile", func() {
				viper.Set("logfile", filepath.Join(tmpDir, "foo.log"))
				Expect(cmd.ExecuteContext(context.TODO())).To(Succeed())
				_, err := os.Stat(filepath.Join(tmpDir, "foo.log"))
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
