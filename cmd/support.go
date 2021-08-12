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
	promptErrorMessage   = "Prompt Failed, Please Try re-running support command."
)

var supportCmd = &cobra.Command{
	Use:   "support",
	Short: "Submits a support request",
	Long: `This interactive command will generate a URL; based on user input which can then be used to create a Red Hat Support Ticket.
	This command can be used when you'd like assistance from Red Hat Support when attempting to pass your certification checks. `,
	RunE: func(cmd *cobra.Command, args []string) error {

		certProjectTypeLabel := promptui.Select{
			Label: "Select a Certification Project Type",
			Items: []string{"Container Image", "Operator Bundle Image"},
		}

		_, certProjectTypeValue, err := certProjectTypeLabel.Run()

		if err != nil {
			fmt.Println(promptErrorMessage)
			return err
		}

		log.Debugf("certification project type: %s", certProjectTypeValue)

		certProjectIDLabel := promptui.Prompt{
			Label: "Please Enter Connect Certification Project ID",
			Validate: func(s string) error {
				isLegacy, _ := regexp.MatchString(`^p.*`, s)
				if isLegacy {
					return errors.New("please remove leading character p from project id")
				}

				isOSPID, _ := regexp.MatchString(`^ospid-.*`, s)
				if isOSPID {
					return errors.New("please remove leading character ospid- from project id")
				}

				return nil
			},
		}

		certProjectIDValue, err := certProjectIDLabel.Run()

		if err != nil {
			fmt.Println(promptErrorMessage)
			return err
		}

		log.Debugf("certification project id: %s", certProjectIDValue)

		pullRequestURLLabel := promptui.Prompt{
			Label: "Please Enter Your Pull Request URL",
		}

		pullRequestURLValue, err := pullRequestURLLabel.Run()

		if err != nil {
			fmt.Println(promptErrorMessage)
			return err
		}

		log.Debugf("pull request url: %s", pullRequestURLValue)

		// building and encoding query params
		queryParams := url.Values{}
		queryParams.Add(typeParam, typeValue)
		queryParams.Add(sourceParam, sourceValue)
		queryParams.Add(certProjectTypeParam, certProjectTypeValue)
		queryParams.Add(certProjectIDParam, certProjectIDValue)
		queryParams.Add(pullRequestURLParam, pullRequestURLValue)

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
