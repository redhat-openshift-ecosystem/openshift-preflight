package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/cli"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/lib"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/viper"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Check Container Command", func() {
	BeforeEach(createAndCleanupDirForArtifactsAndLogs)

	Context("when running the check container subcommand", func() {
		Context("With all of the required parameters", func() {
			It("should reach the core logic, but throw an error because of the placeholder values for the container image", func() {
				_, err := executeCommand(checkContainerCmd(mockRunPreflight), "example.com/example/image:mytag")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("When validating check container arguments and flags", func() {
		Context("and the user provided more than 1 positional arg", func() {
			It("should fail to run", func() {
				_, err := executeCommand(checkContainerCmd(mockRunPreflight), "foo", "bar")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("and the user provided less than 1 positional arg", func() {
			It("should fail to run", func() {
				_, err := executeCommand(checkContainerCmd(mockRunPreflight))
				Expect(err).To(HaveOccurred())
			})
		})

		DescribeTable("and the user has enabled the submit flag",
			func(errString string, args []string) {
				out, err := executeCommand(checkContainerCmd(mockRunPreflight), args...)
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

					err := checkContainerPositionalArgs(checkContainerCmd(mockRunPreflight), []string{"foo"})
					Expect(err).ToNot(HaveOccurred())
					Expect(viper.Instance().GetString("pyxis_api_token")).To(Equal("tokenid"))
					Expect(viper.Instance().GetString("certification_project_id")).To(Equal("certid"))
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
					viper.Instance().AddConfigPath(tempDir)
					DeferCleanup(os.RemoveAll, tempDir)
				})
				It("should still execute with no error", func() {
					// Make sure that we've read the config file
					initConfig()
					submit = true

					err := checkContainerPositionalArgs(checkContainerCmd(mockRunPreflight), []string{"foo"})
					Expect(err).ToNot(HaveOccurred())
					Expect(viper.Instance().GetString("pyxis_api_token")).To(Equal("mytoken"))
					Expect(viper.Instance().GetString("certification_project_id")).To(Equal("mycertid"))
				})
			})
		})
	})

	Context("When validating the certification-project-id flag", func() {
		Context("and the flag is set properly", func() {
			BeforeEach(func() {
				viper.Instance().Set("certification_project_id", "123456789")
				DeferCleanup(viper.Instance().Set, "certification_project_id", "")
			})
			It("should not change the flag value", func() {
				err := validateCertificationProjectID(checkContainerCmd(mockRunPreflight), []string{"foo"})
				Expect(err).ToNot(HaveOccurred())
				Expect(viper.Instance().GetString("certification_project_id")).To(Equal("123456789"))
			})
		})
		Context("and a valid ospid format is provided", func() {
			BeforeEach(func() {
				viper.Instance().Set("certification_project_id", "ospid-123456789")
				DeferCleanup(viper.Instance().Set, "certification_project_id", "")
			})
			It("should strip ospid- from the flag value", func() {
				err := validateCertificationProjectID(checkContainerCmd(mockRunPreflight), []string{"foo"})
				Expect(err).ToNot(HaveOccurred())
				Expect(viper.Instance().GetString("certification_project_id")).To(Equal("123456789"))
			})
		})
		Context("and a legacy format with ospid is provided", func() {
			BeforeEach(func() {
				viper.Instance().Set("certification_project_id", "ospid-62423-f26c346-6cc1dc7fae92")
				DeferCleanup(viper.Instance().Set, "certification_project_id", "")
			})
			It("should throw an error", func() {
				err := validateCertificationProjectID(checkContainerCmd(mockRunPreflight), []string{"foo"})
				Expect(err).To(HaveOccurred())
			})
		})
		Context("and a legacy format without ospid is provided", func() {
			BeforeEach(func() {
				viper.Instance().Set("certification_project_id", "62423-f26c346-6cc1dc7fae92")
				DeferCleanup(viper.Instance().Set, "certification_project_id", "")
			})
			It("should throw an error", func() {
				err := validateCertificationProjectID(checkContainerCmd(mockRunPreflight), []string{"foo"})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("when running the check container subcommand with a logger provided", func() {
		Context("with all of the required parameters", func() {
			It("should reach the core logic, and execute the mocked RunPreflight", func() {
				_, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflight), logr.Discard(), "example.com/example/image:mytag")
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})

func mockRunPreflight(context.Context, func(ctx context.Context) (certification.Results, error), cli.CheckConfig, formatters.ResponseFormatter, lib.ResultWriter, lib.ResultSubmitter) error {
	return nil
}

func mockRunPreflightReturnErr(context.Context, func(ctx context.Context) (certification.Results, error), cli.CheckConfig, formatters.ResponseFormatter, lib.ResultWriter, lib.ResultSubmitter) error {
	return errors.New("random error")
}
