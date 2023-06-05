package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/spf13/cobra"
)

const (
	baseURL                 = "https://connect.redhat.com/support/technology-partner/#/case/new?"
	typeParam               = "type"
	typeValue               = "CERT"
	sourceParam             = "source"
	sourceValue             = "preflight"
	certProjectTypeParam    = "cert_project_type"
	certProjectIDParam      = "cert_project_id"
	pullRequestURLParam     = "pull_request_url"
	operatorBundleImageText = "Operator Bundle Image"
	containerImageText      = "Container Image"
)

var projectTypeMapping = map[string]string{
	"container": containerImageText,
	"operator":  operatorBundleImageText,
}

func supportCmd() *cobra.Command {
	supportCmd := &cobra.Command{
		Use:   "support <operator|container> <your project ID> [pullRequestURL if operator project]",
		Short: "Creates a support request",
		Long:  `Generate a URL that can be used to open a ticket with Red Hat Support if you're having an issue passing certification checks.`,
	}

	supportCmd.AddCommand(supportOperatorCmd())
	supportCmd.AddCommand(supportContainerCmd())

	return supportCmd
}

type supportTextGenerator struct {
	ProjectType    string
	ProjectID      string
	PullRequestURL string
}

func newSupportTextGenerator(ptype, pid, pullReqURL string) (*supportTextGenerator, error) {
	gen := &supportTextGenerator{
		ProjectType:    ptype,
		ProjectID:      pid,
		PullRequestURL: pullReqURL,
	}

	if err := gen.validate(); err != nil {
		return nil, err
	}

	return gen, nil
}

func (g *supportTextGenerator) queryParams() url.Values {
	// base parameters
	qp := url.Values{}
	qp.Add(typeParam, typeValue)
	qp.Add(sourceParam, sourceValue)

	// user parameters
	qp.Add(certProjectTypeParam, projectTypeMapping[g.ProjectType])
	qp.Add(certProjectIDParam, g.ProjectID)

	if g.PullRequestURL != "" {
		qp.Add(pullRequestURLParam, g.PullRequestURL)
	}
	return qp
}

func (g *supportTextGenerator) Generate() string {
	params := g.queryParams()
	return fmt.Sprintf("Create a support ticket by: \n"+
		"\t1. Copying URL: %s\n"+
		"\t2. Paste above URL in a browser\n"+
		"\t3. Login with Red Hat SSO\n"+
		"\t4. Enter an issue summary and description\n"+
		"\t5. Preview and Submit your ticket",
		baseURL+params.Encode())
}

func (g *supportTextGenerator) validate() error {
	if err := projectIDValidation(g.ProjectID); err != nil {
		return err
	}

	if g.ProjectType == "operator" {
		if len(g.PullRequestURL) == 0 {
			return errors.New("a pull request URL is required for operator project support requests")
		}

		if err := pullRequestURLValidation(g.PullRequestURL); err != nil {
			return err
		}
	}

	return nil
}

// pullRequestURLValidation validates urlstr matches expected formats.
// This implements promptui.ValidateFunc.
func pullRequestURLValidation(urlstr string) error {
	_, err := url.ParseRequestURI(urlstr)
	if err != nil {
		return fmt.Errorf("please enter a valid url: including scheme, host, and path to pull request")
	}

	url, err := url.Parse(urlstr)
	if err != nil || url.Scheme == "" || url.Host == "" || url.Path == "" {
		return fmt.Errorf("please enter a valid url: including scheme, host, and path to pull request")
	}

	return nil
}

// projectIDValidation validates id to ensure it conforms with expected formats.
// This implements promptui.ValidateFunc.
func projectIDValidation(id string) error {
	if id == "" {
		return errors.New("please enter a non empty project id")
	}

	isLegacy, _ := regexp.MatchString(`^p.*`, id)
	if isLegacy {
		return errors.New("please remove leading character p from project id")
	}

	isOSPID, _ := regexp.MatchString(`^ospid-.*`, id)
	if isOSPID {
		return errors.New("please remove leading characters ospid- from project id")
	}

	isAlphaNumeric := regexp.MustCompile(`^[a-zA-Z0-9]*$`).MatchString(id)
	if !isAlphaNumeric {
		return errors.New("please remove all special characters from project id")
	}

	return nil
}
