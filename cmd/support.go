package cmd

import (
	"fmt"
	"net/url"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

const (
	baseURL              = "https://connect.dev.redhat.com/support/technology-partner/#/case/new?"
	typeParam            = "type"
	typeValue            = "CERT"
	sourceParam          = "source"
	sourceValue          = "preflight"
	certProjectTypeParam = "cert_project_type"
	certProjectIDParam   = "cert_project_id"
	pullRequestURLParam  = "pull_request_url"
)

var supportCmd = &cobra.Command{
	Use:   "support",
	Short: "Submits a support request",
	Long: `This command will submit a support request to Red Hat along with the logs from the latest Preflight check.
	This command can be used when you'd like assistance from Red Hat Support when attempting to pass your certification checks. `,
	RunE: func(cmd *cobra.Command, args []string) error {

		certProjectTypeLabel := promptui.Select{
			Label: "Select a Certification Project Type",
			Items: []string{"Container Image", "Operator Bundle Image"},
		}

		_, certProjectTypeValue, err := certProjectTypeLabel.Run()

		if err != nil {
			fmt.Println("Prompt Failed, Please Try re-running support command.")
			return err
		}

		fmt.Printf("You Selected:  %q\n", certProjectTypeValue)

		certProjectIDLabel := promptui.Prompt{
			Label: "Please Enter Connect Certification Project ID",
		}

		certProjectIDValue, err := certProjectIDLabel.Run()

		if err != nil {
			fmt.Println("Prompt Failed, Please Try re-running support command.")
			return err
		}

		fmt.Printf("You Entered: %q\n", certProjectIDValue)

		pullRequestURLLabel := promptui.Prompt{
			Label: "Please Enter Your Pull Request URL",
		}

		pullRequestURLValue, err := pullRequestURLLabel.Run()

		if err != nil {
			fmt.Println("Prompt Failed, Please Try re-running support command.")
			return err
		}

		fmt.Printf("You Entered: %q\n", pullRequestURLValue)

		// building and encoding query params
		queryParams := url.Values{}
		queryParams.Add(typeParam, typeValue)
		queryParams.Add(sourceParam, sourceValue)
		queryParams.Add(certProjectTypeParam, certProjectTypeValue)
		queryParams.Add(certProjectIDParam, certProjectIDValue)
		queryParams.Add(pullRequestURLParam, pullRequestURLValue)

		fmt.Printf("URL to Create Support Desk Ticket: %q\n", baseURL+queryParams.Encode())

		return nil
	},
}

func init() {
	rootCmd.AddCommand(supportCmd)
}
