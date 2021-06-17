package main

import (
	"fmt"
	"os"

	"log"

	"github.com/komish/preflight/certification"
	"github.com/komish/preflight/certification/formatters"
	"github.com/komish/preflight/certification/runtime"
)

func main() {
	// log.Println("Running preflight checks")
	runner, err := runtime.NewForConfig(debugConfig)
	if err != nil {
		log.Fatal(err)
	}

	runner.ExecutePolicies()
	results := runner.GetResults()

	formattedResults, err := formatters.GenericJSONFormatter(results)
	if err != nil {
		log.Fatal(err)
	}

	// log.Println("Printing Output")
	fmt.Fprint(os.Stdout, string(formattedResults))
}

// TODO delete me - for testing only
var debugConfig = runtime.Config{
	Image:           "registry.access.redhat.com/ubi8/ubi:sha256:77623387101abefbf83161c7d5a0378379d0424b2244009282acb39d42f1fe13",
	EnabledPolicies: certification.AllPolicies(),
}
