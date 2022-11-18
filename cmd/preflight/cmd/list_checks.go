package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/engine"

	"github.com/spf13/cobra"
)

func listChecksCmd() *cobra.Command {
	listChecksCmd := &cobra.Command{
		Use:   "list-checks",
		Short: "List all checks that will be executed for each policy",
		Long:  "This command will list all checks that preflight uses against an asset by policy type",
		Run:   listChecksRunFunc,
	}
	return listChecksCmd
}

// listChecksRunFunc binds printChecks to cobra's Run function
// definition, passing the cobra command's output as an io.Writer.
func listChecksRunFunc(cmd *cobra.Command, args []string) {
	printChecks(cmd.OutOrStdout())
}

// printChecks writes the formatted check list output to w.
func printChecks(w io.Writer) {
	fmt.Fprintln(w, "These are the available checks for each policy:")
	fmt.Fprintln(w, formattedPolicyBlock("Operator", engine.OperatorPolicy(context.TODO()), "invoked on operator bundles"))
	fmt.Fprintln(w, formattedPolicyBlock("Container", engine.ContainerPolicy(context.TODO()), "invoked on container images"))
	fmt.Fprintln(w, formattedPolicyBlock("Container Root Exception", engine.RootExceptionContainerPolicy(context.TODO()),
		"automatically applied for container images if preflight determines a root exception flag has been added to your Red Hat Connect project"))
	fmt.Fprintln(w, formattedPolicyBlock("Container Scratch Exception", engine.ScratchContainerPolicy(context.TODO()),
		"automatically applied for container checks if preflight determines a scratch exception flag has been added to your Red Hat Connect project"))
}

// formattedPolicyBlock accepts information about the checklist
// and formats it for output.
func formattedPolicyBlock(policyName string, checkList []string, desc string) string {
	title := fmt.Sprintf("[%s Policy]: %s", policyName, desc) // the name in brackets
	list := formatList(checkList)

	return strings.Join([]string{title, list}, "\n")
}

// formatList returns list as a hyphen-prefixed, newline delimited string.
func formatList(list []string) string {
	var s string
	for _, v := range list {
		s += dashPrefix(v) + "\n"
	}

	return s
}

// dashPrefix prefixes string s with a hyphen.
func dashPrefix(s string) string {
	return fmt.Sprintf("- %s", s)
}
