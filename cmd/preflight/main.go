package main

import (
	"log"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/cmd/preflight/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
