package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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
