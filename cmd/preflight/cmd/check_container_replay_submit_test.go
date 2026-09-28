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

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/replay"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/viper"
)

var _ = Describe("replay-submit command wiring", func() {
	BeforeEach(func() {
		viper.Reset()
	})

	// Regression test: registering replaySubmitCmd as a sub-command of
	// checkContainerCmd must NOT reuse the same viper keys checkContainerCmd's
	// own flags are bound to (e.g. "pyxis_api_token", "certification_component_id").
	// Command construction happens once at process startup, so a second
	// viper.BindPFlag call on those keys - even for a never-invoked
	// sub-command - permanently overwrites checkContainerCmd's own binding,
	// breaking `check container --submit` for the lifetime of the process.
	It("does not shadow checkContainerCmd's own pyxis/certification flags", func() {
		root := checkCmd()

		containerCmd, _, err := root.Find([]string{"container"})
		Expect(err).ToNot(HaveOccurred())

		Expect(containerCmd.Flags().Set("pyxis-api-token", "REAL-TOKEN")).To(Succeed())
		Expect(containerCmd.Flags().Set("certification-component-id", "REAL-ID")).To(Succeed())
		Expect(containerCmd.Flags().Set("pyxis-host", "real.pyxis.example.com")).To(Succeed())

		v := viper.Instance()
		Expect(v.GetString("pyxis_api_token")).To(Equal("REAL-TOKEN"),
			"replay-submit's flag registration must not shadow check container's own --pyxis-api-token")
		Expect(v.GetString("certification_component_id")).To(Equal("REAL-ID"),
			"replay-submit's flag registration must not shadow check container's own --certification-component-id")
		Expect(v.GetString("pyxis_host")).To(Equal("real.pyxis.example.com"),
			"replay-submit's flag registration must not shadow check container's own --pyxis-host")
	})

	It("reads its own pyxis/certification flags on the replay-submit sub-command independently", func() {
		root := checkCmd()

		replaySubmit, _, err := root.Find([]string{"container", "replay-submit"})
		Expect(err).ToNot(HaveOccurred())

		Expect(replaySubmit.Flags().Set("pyxis-api-token", "REPLAY-TOKEN")).To(Succeed())
		// Real component IDs from connect.redhat.com's legacy "ospid-<id>" URL
		// format don't contain further dashes, e.g. "ospid-63d1d3d3f8949891ababd53c".
		Expect(replaySubmit.Flags().Set("certification-component-id", "ospid-63d1d3d3f8949891ababd53c")).To(Succeed())

		v := viper.Instance()
		Expect(v.GetString("replay_pyxis_api_token")).To(Equal("REPLAY-TOKEN"))

		// PreRunE performs the "ospid-" normalization.
		Expect(validateReplaySubmitFlags(replaySubmit, nil)).To(Succeed())
		Expect(v.GetString("replay_certification_component_id")).To(Equal("63d1d3d3f8949891ababd53c"))
	})
})

var _ = Describe("validateReplaySubmitFlags", func() {
	var cmd *cobra.Command

	BeforeEach(func() {
		viper.Reset()
		cmd = replaySubmitCmd()
	})

	Context("when --submit was not requested", func() {
		It("returns nil even if pyxis credentials are missing", func() {
			Expect(validateReplaySubmitFlags(cmd, nil)).To(Succeed())
		})
	})

	Context("when --submit was requested", func() {
		BeforeEach(func() {
			Expect(cmd.Flags().Set("submit", "true")).To(Succeed())
		})

		It("errors when the certification component ID is missing", func() {
			Expect(cmd.Flags().Set("pyxis-api-token", "some-token")).To(Succeed())
			err := validateReplaySubmitFlags(cmd, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("certification component ID"))
		})

		It("errors when the pyxis API token is missing", func() {
			Expect(cmd.Flags().Set("certification-component-id", "abc123")).To(Succeed())
			err := validateReplaySubmitFlags(cmd, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("pyxis API Token"))
		})

		It("succeeds when both are present", func() {
			Expect(cmd.Flags().Set("certification-component-id", "abc123")).To(Succeed())
			Expect(cmd.Flags().Set("pyxis-api-token", "some-token")).To(Succeed())
			Expect(validateReplaySubmitFlags(cmd, nil)).To(Succeed())
		})
	})

	Context("when a malformed legacy component ID is provided", func() {
		It("errors before even checking --submit", func() {
			Expect(cmd.Flags().Set("certification-component-id", "ospid-62423-f26c346-6cc1dc7fae92")).To(Succeed())
			err := validateReplaySubmitFlags(cmd, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("improperly formatted"))
		})
	})
})

var _ = Describe("replaySubmitRunE and runReplaySubmitBatch", func() {
	var origReplayRun func(context.Context, replay.Options) (*replay.Summary, error)
	var origReplayRunBatch func(context.Context, string, replay.Options) (*replay.BatchSummary, error)

	BeforeEach(func() {
		viper.Reset()
		origReplayRun = replayRun
		origReplayRunBatch = replayRunBatch
	})

	AfterEach(func() {
		replayRun = origReplayRun
		replayRunBatch = origReplayRunBatch
	})

	Context("when the given path does not exist", func() {
		It("returns an error without attempting to extract anything", func() {
			_, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnNil), logr.Discard(),
				"replay-submit", "/path/does/not/exist")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when the given path is a single artifacts.tar", func() {
		var tarPath string

		BeforeEach(func() {
			dir, err := os.MkdirTemp("", "replay-submit-cli-test-*")
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(os.RemoveAll, dir)

			tarPath = filepath.Join(dir, "artifacts.tar")
			Expect(os.WriteFile(tarPath, []byte("fake tar contents"), 0o644)).To(Succeed())
		})

		It("succeeds when replay.Run succeeds", func() {
			replayRun = func(ctx context.Context, opts replay.Options) (*replay.Summary, error) {
				Expect(opts.ArtifactsTarPath).To(Equal(tarPath))
				return &replay.Summary{ExtractDir: "/tmp/extracted", BasedOnUbiPased: true, Submitted: false}, nil
			}

			_, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnNil), logr.Discard(),
				"replay-submit", tarPath)
			Expect(err).ToNot(HaveOccurred())
		})

		It("propagates an error when replay.Run fails", func() {
			replayRun = func(ctx context.Context, opts replay.Options) (*replay.Summary, error) {
				return nil, errors.New("simulated replay.Run failure")
			}

			_, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnNil), logr.Discard(),
				"replay-submit", tarPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("simulated replay.Run failure"))
		})
	})

	Context("when the given path is a directory (multi-arch)", func() {
		var root string

		BeforeEach(func() {
			dir, err := os.MkdirTemp("", "replay-submit-cli-batch-test-*")
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(os.RemoveAll, dir)
			root = dir
		})

		It("succeeds when every discovered platform succeeds", func() {
			replayRunBatch = func(ctx context.Context, r string, opts replay.Options) (*replay.BatchSummary, error) {
				Expect(r).To(Equal(root))
				return &replay.BatchSummary{
					Platforms: map[string]*replay.Summary{"amd64": {ExtractDir: "/tmp/amd64"}},
					Errors:    map[string]error{},
				}, nil
			}

			_, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnNil), logr.Discard(),
				"replay-submit", root)
			Expect(err).ToNot(HaveOccurred())
		})

		It("propagates a setup error from RunBatch", func() {
			replayRunBatch = func(ctx context.Context, r string, opts replay.Options) (*replay.BatchSummary, error) {
				return nil, errors.New("could not read root")
			}

			_, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnNil), logr.Discard(),
				"replay-submit", root)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not read root"))
		})

		It("returns an aggregated error when some platforms fail", func() {
			replayRunBatch = func(ctx context.Context, r string, opts replay.Options) (*replay.BatchSummary, error) {
				return &replay.BatchSummary{
					Platforms: map[string]*replay.Summary{"amd64": {ExtractDir: "/tmp/amd64"}},
					Errors:    map[string]error{"arm64": errors.New("pyxis unreachable")},
				}, nil
			}

			_, err := executeCommandWithLogger(checkContainerCmd(mockRunPreflightReturnNil), logr.Discard(),
				"replay-submit", root)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("1 of 2"))
		})
	})
})
