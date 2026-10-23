package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/offline"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/viper"
)

// newFlagsOnlyCmd builds a minimal *cobra.Command with just the two flags
// validateSubmitOfflineArtifactsFlags inspects, so tests can drive its
// cmd.Flag(...).Changed logic directly without the full command tree.
func newFlagsOnlyCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "fake"}
	cmd.Flags().String("certification-component-id", "", "")
	cmd.Flags().String("pyxis-api-token", "", "")
	return cmd
}

var _ = Describe("validateSubmitOfflineArtifactsFlags", func() {
	BeforeEach(func() {
		viper.Reset()
	})

	It("errors when the certification component ID is missing", func() {
		viper.Instance().Set("pyxis_api_token", "some-token")
		err := validateSubmitOfflineArtifactsFlags(newFlagsOnlyCmd())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("certification component ID"))
	})

	It("errors when the pyxis API token is missing", func() {
		viper.Instance().Set("certification_component_id", "abc123")
		err := validateSubmitOfflineArtifactsFlags(newFlagsOnlyCmd())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("pyxis API Token"))
	})

	It("succeeds when both are present via viper (e.g. PFLT_* env vars)", func() {
		viper.Instance().Set("certification_component_id", "abc123")
		viper.Instance().Set("pyxis_api_token", "some-token")
		Expect(validateSubmitOfflineArtifactsFlags(newFlagsOnlyCmd())).To(Succeed())
	})

	It("succeeds when both are present via explicitly-changed flags", func() {
		cmd := newFlagsOnlyCmd()
		Expect(cmd.Flags().Set("certification-component-id", "abc123")).To(Succeed())
		Expect(cmd.Flags().Set("pyxis-api-token", "some-token")).To(Succeed())
		viper.Instance().Set("certification_component_id", "abc123")
		viper.Instance().Set("pyxis_api_token", "some-token")
		Expect(validateSubmitOfflineArtifactsFlags(cmd)).To(Succeed())
	})

	It("errors when --docker-config is set, since it's silently ignored otherwise", func() {
		viper.Instance().Set("certification_component_id", "abc123")
		viper.Instance().Set("pyxis_api_token", "some-token")
		viper.Instance().Set("dockerConfig", "/path/to/docker/config.json")
		err := validateSubmitOfflineArtifactsFlags(newFlagsOnlyCmd())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("--docker-config is not supported"))
	})

	It("errors when the component ID flag was explicitly changed to an empty value", func() {
		cmd := newFlagsOnlyCmd()
		Expect(cmd.Flags().Set("certification-component-id", "")).To(Succeed())
		viper.Instance().Set("pyxis_api_token", "some-token")
		err := validateSubmitOfflineArtifactsFlags(cmd)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("certification component ID cannot be empty"))
	})

	It("errors when the pyxis API token flag was explicitly changed to an empty value", func() {
		cmd := newFlagsOnlyCmd()
		Expect(cmd.Flags().Set("pyxis-api-token", "")).To(Succeed())
		viper.Instance().Set("certification_component_id", "abc123")
		err := validateSubmitOfflineArtifactsFlags(cmd)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("pyxis API Token cannot be empty"))
	})

	It("errors when a flag value looks like a missing argument (starts with --)", func() {
		viper.Instance().Set("certification_component_id", "abc123")
		viper.Instance().Set("pyxis_api_token", "--certification-component-id")
		err := validateSubmitOfflineArtifactsFlags(newFlagsOnlyCmd())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("pyxis API token and certification component ID are required"))
	})
})

var _ = Describe("submitOfflineArtifactsRunE", func() {
	var origOfflineRun func(context.Context, offline.Options) (*offline.Summary, error)
	var tarPath string

	BeforeEach(func() {
		viper.Reset()
		origOfflineRun = offlineRun

		dir, err := os.MkdirTemp("", "submit-offline-artifacts-cli-test-*")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(os.RemoveAll, dir)

		tarPath = filepath.Join(dir, "artifacts.tar")
		Expect(os.WriteFile(tarPath, []byte("fake tar contents"), 0o644)).To(Succeed())
	})

	AfterEach(func() {
		offlineRun = origOfflineRun
	})

	It("passes the certification-component-id/pyxis-api-token through to offline.Run and submits unconditionally", func() {
		offlineRun = func(ctx context.Context, opts offline.Options) (*offline.Summary, error) {
			Expect(opts.ArtifactsTarPath).To(Equal(tarPath))
			Expect(opts.CertificationComponentID).To(Equal("abc123"))
			Expect(opts.PyxisAPIToken).To(Equal("some-token"))
			return &offline.Summary{ExtractDir: "/tmp/extracted", BasedOnUbiPassed: true, Submitted: true}, nil
		}

		_, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnNil), logr.Discard(),
			"submit-offline-artifacts", tarPath,
			"--certification-component-id", "abc123",
			"--pyxis-api-token", "some-token")
		Expect(err).ToNot(HaveOccurred())
	})

	It("propagates an error when offline.Run fails", func() {
		offlineRun = func(ctx context.Context, opts offline.Options) (*offline.Summary, error) {
			return nil, errors.New("simulated offline.Run failure")
		}

		_, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnNil), logr.Discard(),
			"submit-offline-artifacts", tarPath,
			"--certification-component-id", "abc123",
			"--pyxis-api-token", "some-token")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("simulated offline.Run failure"))
	})

	It("errors before calling offline.Run when Pyxis credentials are missing", func() {
		offlineRun = func(ctx context.Context, opts offline.Options) (*offline.Summary, error) {
			Fail("offline.Run should not be called when required flags are missing")
			return nil, nil
		}

		_, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnNil), logr.Discard(),
			"submit-offline-artifacts", tarPath)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("certification component ID"))
	})

	It("errors before calling offline.Run when certification-component-id is an improperly formatted legacy value", func() {
		offlineRun = func(ctx context.Context, opts offline.Options) (*offline.Summary, error) {
			Fail("offline.Run should not be called when certification-component-id is malformed")
			return nil, nil
		}

		_, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnNil), logr.Discard(),
			"submit-offline-artifacts", tarPath,
			"--certification-component-id", "ospid-62423-f26c346-6cc1dc7fae92",
			"--pyxis-api-token", "some-token")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("improperly formatted"))
	})

	// submit-offline-artifacts doesn't declare its own Pyxis flags - it
	// inherits checkContainerCmd's persistent ones, so setting them at the
	// parent "container" level must work too.
	It("honors --pyxis-api-token/--certification-component-id set on the parent container command", func() {
		offlineRun = func(ctx context.Context, opts offline.Options) (*offline.Summary, error) {
			Expect(opts.CertificationComponentID).To(Equal("PARENT-ID"))
			Expect(opts.PyxisAPIToken).To(Equal("PARENT-TOKEN"))
			return &offline.Summary{ExtractDir: "/tmp/extracted", BasedOnUbiPassed: true, Submitted: true}, nil
		}

		_, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnNil), logr.Discard(),
			"--certification-component-id", "PARENT-ID",
			"--pyxis-api-token", "PARENT-TOKEN",
			"submit-offline-artifacts", tarPath)
		Expect(err).ToNot(HaveOccurred())
	})
})
