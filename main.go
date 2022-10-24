package main

import (
	"log"

	"github.com/redhat-openshift-ecosystem/preflight/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
