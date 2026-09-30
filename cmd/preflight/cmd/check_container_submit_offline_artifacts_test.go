package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/replay"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/viper"
)

// validateSubmitOfflineArtifactsFlags only reads from viper.Instance() (it
// doesn't inspect cmd/args), so these tests drive it directly via viper
// rather than through cobra flag parsing. Whether the underlying
// --pyxis-api-token/--certification-component-id flags are actually wired up
// correctly (as checkContainerCmd's shared persistent flags, set on either
// the parent or the sub-command) is covered by the submitOfflineArtifactsRunE
// tests below, which exercise the real command tree via Execute().
var _ = Describe("validateSubmitOfflineArtifactsFlags", func() {
	BeforeEach(func() {
		viper.Reset()
	})

	It("errors when the certification component ID is missing", func() {
		viper.Instance().Set("pyxis_api_token", "some-token")
		err := validateSubmitOfflineArtifactsFlags(nil, nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("certification component ID"))
	})

	It("errors when the pyxis API token is missing", func() {
		viper.Instance().Set("certification_component_id", "abc123")
		err := validateSubmitOfflineArtifactsFlags(nil, nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("pyxis API Token"))
	})

	It("succeeds when both are present", func() {
		viper.Instance().Set("certification_component_id", "abc123")
		viper.Instance().Set("pyxis_api_token", "some-token")
		Expect(validateSubmitOfflineArtifactsFlags(nil, nil)).To(Succeed())
	})
})

var _ = Describe("submitOfflineArtifactsRunE", func() {
	var origReplayRun func(context.Context, replay.Options) (*replay.Summary, error)
	var tarPath string

	BeforeEach(func() {
		viper.Reset()
		origReplayRun = replayRun

		dir, err := os.MkdirTemp("", "submit-offline-artifacts-cli-test-*")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(os.RemoveAll, dir)

		tarPath = filepath.Join(dir, "artifacts.tar")
		Expect(os.WriteFile(tarPath, []byte("fake tar contents"), 0o644)).To(Succeed())
	})

	AfterEach(func() {
		replayRun = origReplayRun
	})

	It("passes the certification-component-id/pyxis-api-token through to replay.Run and submits unconditionally", func() {
		replayRun = func(ctx context.Context, opts replay.Options) (*replay.Summary, error) {
			Expect(opts.ArtifactsTarPath).To(Equal(tarPath))
			Expect(opts.CertificationComponentID).To(Equal("abc123"))
			Expect(opts.PyxisAPIToken).To(Equal("some-token"))
			Expect(opts.Submit).To(BeTrue())
			return &replay.Summary{ExtractDir: "/tmp/extracted", BasedOnUbiPased: true, Submitted: true}, nil
		}

		_, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnNil), logr.Discard(),
			"submit-offline-artifacts", tarPath,
			"--certification-component-id", "abc123",
			"--pyxis-api-token", "some-token")
		Expect(err).ToNot(HaveOccurred())
	})

	It("propagates an error when replay.Run fails", func() {
		replayRun = func(ctx context.Context, opts replay.Options) (*replay.Summary, error) {
			return nil, errors.New("simulated replay.Run failure")
		}

		_, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnNil), logr.Discard(),
			"submit-offline-artifacts", tarPath,
			"--certification-component-id", "abc123",
			"--pyxis-api-token", "some-token")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("simulated replay.Run failure"))
	})

	It("errors before calling replay.Run when Pyxis credentials are missing", func() {
		replayRun = func(ctx context.Context, opts replay.Options) (*replay.Summary, error) {
			Fail("replay.Run should not be called when required flags are missing")
			return nil, nil
		}

		_, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnNil), logr.Discard(),
			"submit-offline-artifacts", tarPath)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("certification component ID"))
	})

	// submit-offline-artifacts deliberately does not declare its own
	// --pyxis-api-token/--certification-component-id flags: it inherits
	// checkContainerCmd's persistent ones, so setting them at the parent
	// "container" level (rather than on the sub-command itself) must work
	// too.
	It("honors --pyxis-api-token/--certification-component-id set on the parent container command", func() {
		replayRun = func(ctx context.Context, opts replay.Options) (*replay.Summary, error) {
			Expect(opts.CertificationComponentID).To(Equal("PARENT-ID"))
			Expect(opts.PyxisAPIToken).To(Equal("PARENT-TOKEN"))
			return &replay.Summary{ExtractDir: "/tmp/extracted", BasedOnUbiPased: true, Submitted: true}, nil
		}

		_, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnNil), logr.Discard(),
			"--certification-component-id", "PARENT-ID",
			"--pyxis-api-token", "PARENT-TOKEN",
			"submit-offline-artifacts", tarPath)
		Expect(err).ToNot(HaveOccurred())
	})
})
