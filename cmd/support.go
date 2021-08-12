package cmd

import (
	"errors"
	"fmt"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"strconv"
)

//todo-adam add the query params and static values here
var (
	name = false
)

var supportCmd = &cobra.Command{
	Use:   "support",
	Short: "Submits a support request",
	Long: `This command will submit a support request to Red Hat along with the logs from the latest Preflight checck.
	This command can be used when you'd like assistance from Red Hat Support when attempting to pass your certification checks. `,
	Run: func(cmd *cobra.Command, args []string) {

		//todo-adam update the selector for the command type
		prompt1 := promptui.Select{
			Label: "Select Day",
			Items: []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday",
				"Saturday", "Sunday"},
		}

		_, result, err := prompt1.Run()

		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		fmt.Printf("You choose %q\n", result)

		//todo-adam this would be for text input
		validate := func(input string) error {
			_, err := strconv.ParseFloat(input, 64)
			if err != nil {
				return errors.New("Invalid number")
			}
			return nil
		}

		prompt := promptui.Prompt{
			Label:    "Number",
			Validate: validate,
		}

		result, err = prompt.Run()

		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		fmt.Printf("You choose %q\n", result)
	},
}

func init() {
	rootCmd.AddCommand(supportCmd)
}
