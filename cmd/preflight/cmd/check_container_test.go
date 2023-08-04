package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/cli"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/lib"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

var _ = Describe("Check Container Command", func() {
	BeforeEach(createAndCleanupDirForArtifactsAndLogs)

	When("running the check container subcommand", func() {
		When("with all of the required parameters", func() {
			It("should reach the core logic, but throw an error because of the placeholder values for the container image", func() {
				_, err := executeCommand(checkContainerCmd(mockRunPreflightReturnNil, viper.New()), "example.com/example/image:mytag")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	When("validating check container arguments and flags", func() {
		When("the user provided more than 1 positional arg", func() {
			It("should fail to run", func() {
				_, err := executeCommand(checkContainerCmd(mockRunPreflightReturnNil, viper.New()), "foo", "bar")
				Expect(err).To(HaveOccurred())
			})
		})

		When("the user provided less than 1 positional arg", func() {
			It("should fail to run", func() {
				_, err := executeCommand(checkContainerCmd(mockRunPreflightReturnNil, viper.New()))
				Expect(err).To(HaveOccurred())
			})
		})

		DescribeTable("and the user has enabled the submit flag",
			func(errString string, args []string) {
				out, err := executeCommand(checkContainerCmd(mockRunPreflightReturnNil, viper.New()), args...)
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
			var v *viper.Viper

			BeforeEach(func() {
				v = viper.New()
			})
			When("environment variables are used for certification ID and api token", func() {
				BeforeEach(func() {
					os.Setenv("PFLT_CERTIFICATION_PROJECT_ID", "certid")
					os.Setenv("PFLT_PYXIS_API_TOKEN", "tokenid")
					DeferCleanup(os.Unsetenv, "PFLT_CERTIFICATION_PROJECT_ID")
					DeferCleanup(os.Unsetenv, "PFLT_PYXIS_API_TOKEN")
				})
				It("should still execute with no error", func() {
					initConfig(v)
					submit = true
					err := checkContainerPositionalArgs(checkContainerCmd(mockRunPreflightReturnNil, v), []string{"foo"}, v)
					Expect(err).ToNot(HaveOccurred())
					Expect(v.GetString("pyxis_api_token")).To(Equal("tokenid"))
					Expect(v.GetString("certification_project_id")).To(Equal("certid"))
				})
			})
			When("a config file is used", func() {
				BeforeEach(func() {
					fs := afero.NewMemMapFs()
					v.SetFs(fs)
					config := `pyxis_api_token: mytoken
certification_project_id: mycertid`
					Expect(afero.WriteFile(fs, "/config/config.yaml", bytes.NewBufferString(config).Bytes(), 0o644)).To(Succeed())
					v.AddConfigPath("/config")
				})
				It("should still execute with no error", func() {
					// Make sure that we've read the config file
					initConfig(v)
					submit = true

					err := checkContainerPositionalArgs(checkContainerCmd(mockRunPreflightReturnNil, v), []string{"foo"}, v)
					Expect(err).ToNot(HaveOccurred())
					Expect(v.GetString("pyxis_api_token")).To(Equal("mytoken"))
					Expect(v.GetString("certification_project_id")).To(Equal("mycertid"))
				})
			})
		})
	})

	DescribeTable("validating the certification-project-id flag",
		func(value, expected string, succeed bool) {
			updated, err := validateCertificationProjectID(value)
			if succeed {
				Expect(err).ToNot(HaveOccurred())
				Expect(updated).To(Equal(expected))
				return
			}
			Expect(err).To(HaveOccurred())
		},
		Entry("the flag is set properly should not change value", "123456789", "123456789", true),
		Entry("a valid ospid should strip ospid-", "ospid-123456789", "123456789", true),
		Entry("legacy format with ospid", "ospid-62423-f26c346-6cc1dc7fae92", "", false),
		Entry("legacy format without ospid", "62423-f26c346-6cc1dc7fae92", "", false),
	)

	When("running the check container subcommand with a logger provided", func() {
		When("with all of the required parameters", func() {
			It("should reach the core logic, and execute the mocked RunPreflight", func() {
				_, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnNil, viper.New()), logr.Discard(), "example.com/example/image:mytag")
				Expect(err).ToNot(HaveOccurred())
			})
		})
		When("with all of the required parameters with error mocked", func() {
			It("should reach the core logic, and execute the mocked RunPreflight and return error", func() {
				_, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnErr, viper.New()), logr.Discard(), "example.com/example/image:mytag")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	When("running the check container subcommand with a offline config provided", func() {
		var v *viper.Viper

		BeforeEach(func() {
			v = viper.New()
		})
		When("with all of the required parameters", func() {
			BeforeEach(func() {
				tmpDir, err := os.MkdirTemp("", "preflight-submit-test-*")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(os.RemoveAll, tmpDir)

				// creating test files on in the tmpDir so the tar function has files to tar
				f1, err := os.Create(filepath.Join(tmpDir, "test-file-1.json"))
				Expect(err).ToNot(HaveOccurred())
				defer f1.Close()

				f2, err := os.Create(filepath.Join(tmpDir, "test-file-1.json"))
				Expect(err).ToNot(HaveOccurred())
				defer f2.Close()

				v.Set("artifacts", tmpDir)
				v.Set("offline", true)
			})
			It("should reach core logic, and the additional offline logic", func() {
				out, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnNil, v), logr.Discard(), "example.com/example/image:mytag")
				Expect(err).ToNot(HaveOccurred())
				Expect(out).ToNot(BeNil())
			})
		})
		When("an existing artifacts.tar already on disk", func() {
			BeforeEach(func() {
				tmpDir, err := os.MkdirTemp("", "preflight-submit-test-*")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(os.RemoveAll, tmpDir)

				// creating test files on in the tmpDir so the tar function has files to tar
				f1, err := os.Create(filepath.Join(tmpDir, "test-file-1.json"))
				Expect(err).ToNot(HaveOccurred())
				defer f1.Close()

				f2, err := os.Create(filepath.Join(tmpDir, "test-file-1.json"))
				Expect(err).ToNot(HaveOccurred())
				defer f2.Close()

				// creating a tar file to mimic a user re-running check container for a second time
				f3, err := os.Create(filepath.Join(tmpDir, check.DefaultArtifactsTarFileName))
				Expect(err).ToNot(HaveOccurred())
				defer f3.Close()

				v.Set("artifacts", tmpDir)
				v.Set("offline", true)
			})
			It("should reach the additional offline logic, and remove existing tar file", func() {
				out, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnNil, v), logr.Discard(), "example.com/example/image:mytag")
				Expect(err).ToNot(HaveOccurred())
				Expect(out).ToNot(BeNil())
			})
		})
	})

	When("artifactsTar is called directly", func() {
		When("the src does not exist", func() {
			It("should get an unable to tar files error", func() {
				err := artifactsTar(context.Background(), "", nil)
				Expect(err).To(HaveOccurred())
			})
		})
		When("a bad writer is passed", func() {
			It("should get an error with trying to write the header", func() {
				err := artifactsTar(context.Background(), ".", errWriter(0))
				Expect(err).To(HaveOccurred())
			})
		})
		When("a src has no permissions", func() {
			var tmpDir string
			var err error
			var buf bytes.Buffer
			JustBeforeEach(func() {
				tmpDir, err = os.MkdirTemp("", "preflight-submit-test-*")
				Expect(err).ToNot(HaveOccurred())
				err = os.Chmod(tmpDir, 0o000)
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(os.RemoveAll, tmpDir)
			})
			It("should get an error with trying to read directory", func() {
				err := artifactsTar(context.Background(), tmpDir, &buf)
				Expect(err).To(HaveOccurred())
			})
		})
		When("and src only contains another directory", func() {
			var tmpDir string
			var err error
			var buf bytes.Buffer
			JustBeforeEach(func() {
				tmpDir, err = os.MkdirTemp("", "preflight-submit-test-*")
				Expect(err).ToNot(HaveOccurred())
				err = os.Mkdir(filepath.Join(tmpDir, "test"), 0o755)
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(os.RemoveAll, tmpDir)
			})
			It("should continue and not tar any files", func() {
				err := artifactsTar(context.Background(), tmpDir, &buf)
				Expect(err).To(BeNil())
			})
		})
	})
})

func mockRunPreflightReturnNil(context.Context, func(ctx context.Context) (certification.Results, error), cli.CheckConfig, formatters.ResponseFormatter, lib.ResultWriter, lib.ResultSubmitter, string) error {
	return nil
}

func mockRunPreflightReturnErr(context.Context, func(ctx context.Context) (certification.Results, error), cli.CheckConfig, formatters.ResponseFormatter, lib.ResultWriter, lib.ResultSubmitter, string) error {
	return errors.New("random error")
}
