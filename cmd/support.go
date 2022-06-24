package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/manifoldco/promptui"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	baseURL              = "https://connect.redhat.com/support/technology-partner/#/case/new?"
	typeParam            = "type"
	typeValue            = "CERT"
	sourceParam          = "source"
	sourceValue          = "preflight"
	certProjectTypeParam = "cert_project_type"
	certProjectIDParam   = "cert_project_id"
	pullRequestURLParam  = "pull_request_url"
	operatorBundleImage  = "Operator Bundle Image"
	containerImage       = "Container Image"
)

var supportCmd = &cobra.Command{
	Use:   "support",
	Short: "Submits a support request",
	Long: `This interactive command will generate a URL; based on user input which can then be used to create a Red Hat Support Ticket.
	This command can be used when you'd like assistance from Red Hat Support when attempting to pass your certification checks. `,
	RunE: supportRunE,
}

func supportRunE(cmd *cobra.Command, args []string) error {
	_, projectType, err := projectTypeSelect().Run()
	if err != nil {
		return fmt.Errorf("project type prompt failed, please try re-running support command")
	}

	log.Debugf("certification project type: %s", projectType)

	projectID, err := projectIDPrompt().Run()
	if err != nil {
		return fmt.Errorf("project ID prompt failed, please try re-running support command")
	}

	log.Debugf("certification project id: %s", projectID)

	// building and encoding query params

	// checking project type to see if we need to add additional query params
	var pullRequestURL string
	if projectType == operatorBundleImage {

		pullRequestURL, err = pullRequestURLPrompt().Run()
		if err != nil {
			return fmt.Errorf("pull request URL prompt failed, please try re-running support command")
		}

		log.Debugf("pull request url: %s", pullRequestURL)
	}

	// pullRequestURL emptiness is handled internally.
	queryParams := queryParams(projectType, projectID, pullRequestURL)

	fmt.Fprintln(cmd.OutOrStdout(), supportInstructions(baseURL, queryParams))

	return nil
}

func init() {
	rootCmd.AddCommand(supportCmd)
}

// queryParams builds out url.Values with base parameters, and those passed in as values.
// optionalPullRequestURL can be empty, and if so, will not be included.
func queryParams(projectType, projectID, optionalPullRequestURL string) url.Values {
	// base parameters
	qp := url.Values{}
	qp.Add(typeParam, typeValue)
	qp.Add(sourceParam, sourceValue)
	// user parameters
	qp.Add(certProjectTypeParam, projectType)
	qp.Add(certProjectIDParam, projectID)
	if optionalPullRequestURL != "" {
		qp.Add(pullRequestURLParam, optionalPullRequestURL)
	}
	return qp
}

// supportInstructions returns a string containing the steps to get support
func supportInstructions(baseURL string, queryParams url.Values) string {
	return fmt.Sprintf("Create a support ticket by: \n"+
		"\t1. Copying URL: %s\n"+
		"\t2. Paste above URL in a browser\n"+
		"\t3. Login with Red Hat SSO\n"+
		"\t4. Enter an issue summary and description\n"+
		"\t5. Preview and Submit your ticket",
		baseURL+queryParams.Encode())
}

// pullRequestURLPrompt returns the promptui.Prompt for receiving the pull request URL
// from the user.
func pullRequestURLPrompt() *promptui.Prompt {
	return &promptui.Prompt{
		Label: "Please Enter Your Pull Request URL",

		// validate makes sure that the url entered has a valid scheme, host and path to the pull request
		Validate: pullRequestURLValidation,
	}
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

// projectIDPrompt returns a promptui.Prompt that receives and validates the user's Connect
// certification project ID.
func projectIDPrompt() *promptui.Prompt {
	return &promptui.Prompt{
		Label: "Please Enter Connect Certification Project ID",

		// validate makes sure that the project id is not blank, does not contain special characters,
		// and is in the proper format
		Validate: projectIDValidation,
	}
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
		return errors.New("please remove leading character ospid- from project id")
	}

	isAlphaNumeric := regexp.MustCompile(`^[a-zA-Z0-9]*$`).MatchString(id)
	if !isAlphaNumeric {
		return errors.New("please remove all special characters from project id")
	}

	return nil
}

// projectTypeSelect returns a promptui.Select allowing a user to select either a container
// or operator project type.
func projectTypeSelect() *promptui.Select {
	return &promptui.Select{
		Label: "Select a Certification Project Type",
		Items: []string{containerImage, operatorBundleImage},
	}
}
