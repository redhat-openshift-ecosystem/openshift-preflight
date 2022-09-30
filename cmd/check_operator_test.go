package cmd

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)

var _ = Describe("Check Operator", func() {
	Context("when running the check operator subcommand", func() {
		BeforeEach(createAndCleanupDirForArtifactsAndLogs)
		Context("without the operator bundle image being provided", func() {
			It("should return an error", func() {
				_, err := executeCommand(checkOperatorCmd())
				Expect(err).To(HaveOccurred())
			})
		})

		Context("without having set the KUBECONFIG environment variable", func() {
			BeforeEach(func() {
				if val, isSet := os.LookupEnv("KUBECONFIG"); isSet {
					DeferCleanup(os.Setenv, "KUBECONFIG", val)
				}
				os.Unsetenv("KUBECONFIG")
			})
			It("should return an error", func() {
				out, err := executeCommand(checkOperatorCmd(), "quay.io/example/image:mytag")
				Expect(err).To(HaveOccurred())
				Expect(out).To(ContainSubstring("KUBECONFIG could not"))
			})
		})

		Context("without having set the PFLT_INDEXIMAGE environment variable", func() {
			BeforeEach(func() {
				if val, isSet := os.LookupEnv("PFLT_INDEXIMAGE"); isSet {
					DeferCleanup(os.Setenv, "PFLT_INDEXIMAGE", val)
				}
				os.Unsetenv("PFLT_INDEXIMAGE")
				if val, isSet := os.LookupEnv("KUBECONFIG"); isSet {
					DeferCleanup(os.Setenv, "KUBECONFIG", val)
				} else {
					DeferCleanup(os.Unsetenv, "KUBECONFIG")
				}
				os.Setenv("KUBECONFIG", "foo")
			})
			It("should return an error", func() {
				out, err := executeCommand(checkOperatorCmd(), "quay.io/example/image:mytag")
				Expect(err).To(HaveOccurred())
				Expect(out).To(ContainSubstring("PFLT_INDEXIMAGE could not"))
			})
		})

		Context("With all of the required parameters", func() {
			BeforeEach(func() {
				DeferCleanup(viper.Set, "indexImage", viper.GetString("indexImage"))
				viper.Set("indexImage", "foo")
				if val, isSet := os.LookupEnv("KUBECONFIG"); isSet {
					DeferCleanup(os.Setenv, "KUBECONFIG", val)
				} else {
					DeferCleanup(os.Unsetenv, "KUBECONFIG")
				}
				os.Setenv("KUBECONFIG", "foo")
			})
			It("should reach the core logic, but throw an error because of the placeholder values", func() {
				_, err := executeCommand(checkOperatorCmd(), "quay.io/example/image:mytag")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("When checking for required environment variables", func() {
		Context("specifically, KUBECONFIG", func() {
			BeforeEach(func() {
				if val, isSet := os.LookupEnv("KUBECONIFG"); isSet {
					DeferCleanup(os.Setenv, "KUBECONFIG", val)
				} else {
					DeferCleanup(os.Unsetenv, "KUBECONFIG")
				}
				os.Setenv("KUBECONFIG", "foo")
			})
			It("should not encounter an error if the value is set", func() {
				err := ensureKubeconfigIsSet()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should encounter an error if the value is not set", func() {
				os.Unsetenv("KUBECONFIG")
				err := ensureKubeconfigIsSet()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("specifically, PFLT_INDEXIMAGE", func() {
			BeforeEach(func() {
				DeferCleanup(viper.Set, "indexImage", viper.GetString("indexImage"))
				viper.Set("indexImage", "foo")
			})
			It("should not encounter an error if the value is set", func() {
				err := ensureIndexImageConfigIsSet()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should encounter an error if the value is not set", func() {
				viper.Set("indexImage", "")
				err := ensureIndexImageConfigIsSet()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("When testing positional arg parsing", func() {
		// failure cases are tested earlier in this file by running executeCommand.
		// This tests the success case using the standalone function in order
		// to prevent trying to run the entire RunE func in previous cases.
		posArgs := []string{"firstparam"}
		BeforeEach(func() {
			DeferCleanup(viper.Set, "indexImage", viper.GetString("indexImage"))
			viper.Set("indexImage", "foo")
			if val, isSet := os.LookupEnv("KUBECONIFG"); isSet {
				DeferCleanup(os.Setenv, "KUBECONFIG", val)
			} else {
				DeferCleanup(os.Unsetenv, "KUBECONFIG")
			}
			os.Setenv("KUBECONFIG", "foo")
		})

		It("should succeed when all positional arg constraints and environment constraints are correct", func() {
			err := checkOperatorPositionalArgs(checkOperatorCmd(), posArgs)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
