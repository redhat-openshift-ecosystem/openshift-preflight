package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/cmd/preflight/cmd"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/cli"
)

func main() {
	//coverage:ignore
	if err := cmd.Execute(); err != nil {
		//coverage:ignore
		switch {
		case errors.Is(err, &cli.ChecksErroredError{}):
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		case errors.Is(err, &cli.ChecksFailedError{}):
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		default:
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
