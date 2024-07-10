package cmd

import (
	"context"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/engine"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("list checks subcommand", func() {
	Context("When formatting check lists for print", func() {
		testList := []string{"foo", "bar", "baz"}
		It("should have the same number of items", func() {
			res := formatList(testList)
			resSplit := strings.Split(res, "\n")
			Expect(len(resSplit)).To(Equal(len(testList) + 1)) // account for newline at the end.
		})
	})

	Context("When calling dashPrefix on an input string", func() {
		inputString := "foo"
		It("should be prepended with a hyphen and a space", func() {
			res := dashPrefix(inputString)
			Expect(strings.HasPrefix(res, "- ")).To(BeTrue())
		})
	})

	Context("Preparing a policy block for print", func() {
		t := "MyTitle"
		d := "MyDescription"
		c := []string{"this", "that"}
		expected := "[MyTitle Policy]: MyDescription\n- this\n- that\n"
		It("Should return a string in an expected format", func() {
			res := formattedPolicyBlock(t, c, d)
			Expect(res).To(Equal(expected))
		})
	})

	Context("Printing checks", func() {
		It("should always contain the container policy", func() {
			expected := formatList(engine.ContainerPolicy(context.TODO()))
			buf := strings.Builder{}
			printChecks(&buf)

			Expect(buf.String()).To(ContainSubstring(expected))
		})

		It("should always contain the operator policy", func() {
			expected := formatList(engine.OperatorPolicy(context.TODO()))
			buf := strings.Builder{}
			printChecks(&buf)

			Expect(buf.String()).To(ContainSubstring(expected))
		})

		It("should always contain the root exception policy", func() {
			expected := formatList(engine.RootExceptionContainerPolicy(context.TODO()))
			buf := strings.Builder{}
			printChecks(&buf)

			Expect(buf.String()).To(ContainSubstring(expected))
		})

		It("should always contain the scratch exception policy", func() {
			expected := formatList(engine.ScratchNonRootContainerPolicy(context.TODO()))
			buf := strings.Builder{}
			printChecks(&buf)

			Expect(buf.String()).To(ContainSubstring(expected))
		})
	})

	Context("When executing the cobra command", func() {
		BeforeEach(createAndCleanupDirForArtifactsAndLogs)
		It("should contain output equivalent to printChecks", func() {
			// get the expected result
			buf := strings.Builder{}
			printChecks(&buf)
			expected := buf.String()

			// Run the command. Because we bind this command to the
			// root command in init, we must pass rootCmd to executeCommand.
			out, err := executeCommand(listChecksCmd())
			Expect(len(out) > 0).To(BeTrue())

			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(ContainSubstring(expected))
		})
	})
})
