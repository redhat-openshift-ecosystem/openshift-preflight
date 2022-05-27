package cmd

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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
