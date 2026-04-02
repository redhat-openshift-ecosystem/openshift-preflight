package main

import (
	"log"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/cmd/preflight/cmd"
)

func main() {
	//coverage:ignore
	if err := cmd.Execute(); err != nil {
		//coverage:ignore
		log.Fatal(err)
	}
}
