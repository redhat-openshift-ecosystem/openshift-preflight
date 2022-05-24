package cmd

import (
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
	PreRun: preRunConfig,
	RunE: func(cmd *cobra.Command, args []string) error {
		certProjectTypeLabel := promptui.Select{
			Label: "Select a Certification Project Type",
			Items: []string{containerImage, operatorBundleImage},
		}

		_, certProjectTypeValue, err := certProjectTypeLabel.Run()
		if err != nil {
			return ErrSupportCmdPromptFailed
		}

		log.Debugf("certification project type: %s", certProjectTypeValue)

		certProjectIDLabel := promptui.Prompt{
			Label: "Please Enter Connect Certification Project ID",

			// validate makes sure that the project id is not blank, does not contain special characters,
			// and is in the proper format
			Validate: func(s string) error {
				if s == "" {
					return ErrEmptyProjectID
				}

				isLegacy, _ := regexp.MatchString(`^p.*`, s)
				if isLegacy {
					return ErrRemovePFromProjectID
				}

				isOSPID, _ := regexp.MatchString(`^ospid-.*`, s)
				if isOSPID {
					return ErrRemoveOSPIDFromProjectID
				}

				isAlphaNumeric := regexp.MustCompile(`^[a-zA-Z0-9]*$`).MatchString(s)
				if !isAlphaNumeric {
					return ErrRemoveSpecialCharFromProjectID
				}

				return nil
			},
		}

		certProjectIDValue, err := certProjectIDLabel.Run()
		if err != nil {
			return ErrSupportCmdPromptFailed
		}

		log.Debugf("certification project id: %s", certProjectIDValue)

		// building and encoding query params
		queryParams := url.Values{}
		queryParams.Add(typeParam, typeValue)
		queryParams.Add(sourceParam, sourceValue)
		queryParams.Add(certProjectTypeParam, certProjectTypeValue)
		queryParams.Add(certProjectIDParam, certProjectIDValue)

		// checking project type to see if we need to add additional query params
		if certProjectTypeValue == operatorBundleImage {
			pullRequestURLLabel := promptui.Prompt{
				Label: "Please Enter Your Pull Request URL",

				// validate makes sure that the url entered has a valid scheme, host and path to the pull request
				Validate: func(s string) error {
					_, err := url.ParseRequestURI(s)
					if err != nil {
						return ErrPullRequestURL
					}

					url, err := url.Parse(s)
					if err != nil || url.Scheme == "" || url.Host == "" || url.Path == "" {
						return ErrPullRequestURL
					}

					return nil
				},
			}

			pullRequestURLValue, err := pullRequestURLLabel.Run()
			if err != nil {
				return ErrSupportCmdPromptFailed
			}

			log.Debugf("pull request url: %s", pullRequestURLValue)

			queryParams.Add(pullRequestURLParam, pullRequestURLValue)
		}

		fmt.Printf("Create a support ticket by: \n"+
			"\t1. Copying URL: %s\n"+
			"\t2. Paste above URL in a browser\n"+
			"\t3. Login with Red Hat SSO\n"+
			"\t4. Enter an issue summary and description\n"+
			"\t5. Preview and Submit your ticket\n", baseURL+queryParams.Encode())

		return nil
	},
}

func init() {
	rootCmd.AddCommand(supportCmd)
}
