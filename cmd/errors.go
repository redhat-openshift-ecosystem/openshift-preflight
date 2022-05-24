package cmd

import "errors"

var (
	ErrInsufficientPosArguments       = errors.New("not enough positional arguments")
	ErrEmptyProjectID                 = errors.New("please enter a non empty project id")
	ErrRetrievingProject              = errors.New("could not retrieve project")
	ErrIndexImageUndefined            = errors.New("no environment variable PFLT_INDEXIMAGE could be found")
	ErrSupportCmdPromptFailed         = errors.New("prompt failed, please try re-running support command")
	ErrRemovePFromProjectID           = errors.New("please remove leading character p from project id")
	ErrRemoveOSPIDFromProjectID       = errors.New("please remove leading character ospid- from project id")
	ErrRemoveSpecialCharFromProjectID = errors.New("please remove all special characters from project id")
	ErrPullRequestURL                 = errors.New("please enter a valid url: including scheme, host, and path to pull request")
	ErrSubmittingToPyxis              = errors.New("unable to submit results to Red Hat")
	ErrNoKubeconfig                   = errors.New("no environment variable KUBECONFIG could be found")
)
