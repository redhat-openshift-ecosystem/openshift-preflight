package main

import (
	"errors"
	"log"
	"os"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/cmd/preflight/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		if errors.Is(err, &cmd.ChecksErroredError{}) {
			log.Println(err)
			os.Exit(1)
		} else if errors.Is(err, &cmd.ChecksFailedError{}) {
			log.Println(err)
			os.Exit(2)
		} else {
			log.Fatal(err)
		}
	}
}
