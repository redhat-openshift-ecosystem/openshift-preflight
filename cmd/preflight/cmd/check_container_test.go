package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/lib"
)

var _ = Describe("Check Container Command", func() {
	BeforeEach(createAndCleanupDirForArtifactsAndLogs)

	Context("when running the check container subcommand", func() {
		Context("With all of the required parameters", func() {
			It("should reach the core logic, but throw an error because of the placeholder values for the container image", func() {
				_, err := executeCommand(checkContainerCmd(), "example.com/example/image:mytag")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("When validating check container arguments and flags", func() {
		Context("and the user provided more than 1 positional arg", func() {
			It("should fail to run", func() {
				_, err := executeCommand(checkContainerCmd(), "foo", "bar")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("and the user provided less than 1 positional arg", func() {
			It("should fail to run", func() {
				_, err := executeCommand(checkContainerCmd())
				Expect(err).To(HaveOccurred())
			})
		})

		DescribeTable("and the user has enabled the submit flag",
			func(errString string, args []string) {
				out, err := executeCommand(checkContainerCmd(), args...)
				Expect(err).To(HaveOccurred())
				Expect(out).To(ContainSubstring(errString))
			},
			Entry("certification-project-id and pyxis-api-token are not supplied", "certification Project ID must be specified when --submit is present", []string{"--submit", "foo"}),
			Entry("pyxis-api-token is not supplied", "pyxis API Token must be specified when --submit is present", []string{"foo", "--submit", "--certification-project-id=fooid"}),
			Entry("certification-project-id is not supplied", "certification Project ID must be specified when --submit is present", []string{"--submit", "foo", "--pyxis-api-token=footoken"}),
			Entry("pyxis-api-token flag is present but empty because of '='", "cannot be empty when --submit is present", []string{"foo", "--submit", "--certification-project-id=fooid", "--pyxis-api-token="}),
			Entry("certification-project-id flag is present but empty because of '='", "cannot be empty when --submit is present", []string{"foo", "--submit", "--certification-project-id=", "--pyxis-api-token=footoken"}),
			Entry("submit is passed after empty api token", "pyxis API token and certification ID are required when --submit is present", []string{"foo", "--certification-project-id=fooid", "--pyxis-api-token", "--submit"}),
			Entry("submit is passed with explicit value after empty api token", "pyxis API token and certification ID are required when --submit is present", []string{"foo", "--certification-project-id=fooid", "--pyxis-api-token", "--submit=true"}),
			Entry("submit is passed and insecure is specified", "if any flags in the group [submit insecure] are set", []string{"foo", "--submit", "--insecure", "--certification-project-id=fooid", "--pyxis-api-token=footoken"}),
		)

		When("the user enables the submit flag", func() {
			When("environment variables are used for certification ID and api token", func() {
				BeforeEach(func() {
					os.Setenv("PFLT_CERTIFICATION_PROJECT_ID", "certid")
					os.Setenv("PFLT_PYXIS_API_TOKEN", "tokenid")
					DeferCleanup(os.Unsetenv, "PFLT_CERTIFICATION_PROJECT_ID")
					DeferCleanup(os.Unsetenv, "PFLT_PYXIS_API_TOKEN")
				})
				It("should still execute with no error", func() {
					submit = true

					err := checkContainerPositionalArgs(checkContainerCmd(), []string{"foo"})
					Expect(err).ToNot(HaveOccurred())
					Expect(viper.GetString("pyxis_api_token")).To(Equal("tokenid"))
					Expect(viper.GetString("certification_project_id")).To(Equal("certid"))
				})
			})
			When("a config file is used", func() {
				BeforeEach(func() {
					config := `pyxis_api_token: mytoken
certification_project_id: mycertid`
					tempDir, err := os.MkdirTemp("", "check-container-submit-*")
					Expect(err).ToNot(HaveOccurred())
					err = os.WriteFile(filepath.Join(tempDir, "config.yaml"), bytes.NewBufferString(config).Bytes(), 0o644)
					Expect(err).ToNot(HaveOccurred())
					viper.AddConfigPath(tempDir)
					DeferCleanup(os.RemoveAll, tempDir)
				})
				It("should still execute with no error", func() {
					// Make sure that we've read the config file
					initConfig()
					submit = true

					err := checkContainerPositionalArgs(checkContainerCmd(), []string{"foo"})
					Expect(err).ToNot(HaveOccurred())
					Expect(viper.GetString("pyxis_api_token")).To(Equal("mytoken"))
					Expect(viper.GetString("certification_project_id")).To(Equal("mycertid"))
				})
			})
		})
	})

	Context("When validating the certification-project-id flag", func() {
		Context("and the flag is set properly", func() {
			BeforeEach(func() {
				viper.Set("certification_project_id", "123456789")
			})
			It("should not change the flag value", func() {
				err := validateCertificationProjectID(checkContainerCmd(), []string{"foo"})
				Expect(err).ToNot(HaveOccurred())
				Expect(viper.GetString("certification_project_id")).To(Equal("123456789"))
			})
		})
		Context("and a valid ospid format is provided", func() {
			BeforeEach(func() {
				viper.Set("certification_project_id", "ospid-123456789")
			})
			It("should strip ospid- from the flag value", func() {
				err := validateCertificationProjectID(checkContainerCmd(), []string{"foo"})
				Expect(err).ToNot(HaveOccurred())
				Expect(viper.GetString("certification_project_id")).To(Equal("123456789"))
			})
		})
		Context("and a legacy format with ospid is provided", func() {
			BeforeEach(func() {
				viper.Set("certification_project_id", "ospid-62423-f26c346-6cc1dc7fae92")
			})
			It("should throw an error", func() {
				err := validateCertificationProjectID(checkContainerCmd(), []string{"foo"})
				Expect(err).To(HaveOccurred())
			})
		})
		Context("and a legacy format without ospid is provided", func() {
			BeforeEach(func() {
				viper.Set("certification_project_id", "62423-f26c346-6cc1dc7fae92")
			})
			It("should throw an error", func() {
				err := validateCertificationProjectID(checkContainerCmd(), []string{"foo"})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("When instantiating a checkContainerRunner", func() {
		var cfg *runtime.Config

		Context("and the user did not pass the submit flag", func() {
			var origSubmitValue bool
			BeforeEach(func() {
				origSubmitValue = submit
				submit = false

				cfg = &runtime.Config{
					Image:          "quay.io/example/foo:latest",
					ResponseFormat: formatters.DefaultFormat,
				}
			})

			AfterEach(func() {
				submit = origSubmitValue
			})
			It("should return a NoopSubmitter ResultSubmitter", func() {
				runner, err := lib.NewCheckContainerRunner(context.TODO(), cfg, false)
				Expect(err).ToNot(HaveOccurred())
				_, rsIsCorrectType := runner.Rs.(*lib.NoopSubmitter)
				Expect(rsIsCorrectType).To(BeTrue())
			})
		})
	})
})
