package e2e

import (
	"fmt"
	"io"
	"os/exec"
)

func preflightHelp() string {
	cmd := exec.Command("preflight")
	output, err := cmd.Output()

	if err != nil {
		fmt.Println("error:", err)
	} else {
		//#fmt.Printf("output: %s \n\n\n%s", string(output), string(want))
	}
	return string(output)
}

func preflightCheck() string {
	cmd := exec.Command("preflight", "check")
	output, err := cmd.Output()

	if err != nil {
		fmt.Println("error:", err)
	}
	return string(output)
}

func preflightCheckContainer() string {
	cmd := exec.Command("preflight", "check", "container")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + string(output))
	} else {
		fmt.Printf("output: %s \n\n\n", string(output))

	}
	return string(output)
}

func preflightCheckOperator() string {
	cmd := exec.Command("preflight", "check", "operator")
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Printf("output: %s \n\n\n", string(output))
	}
	return string(output)
}

func preflightSupport() string {
	cmd := exec.Command("preflight", "support")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println("error:", err)
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, "\ntest\nhttps://github.com/redhat-openshift-ecosystem/openshift-preflight/pull/267\n")
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Printf("%s\n", out)

	//output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Printf("output: %s \n\n\n", string(out))
	}
	return string(out)
}

func preflightContainerRun() string {

	cmd := exec.Command("preflight", "check", "container", "quay.io/komish/preflight-test-container-passes:latest")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + string(output))
	} else {
		fmt.Printf("output: %s \n\n\n", string(output))

	}
	return string(output)
}

func preflightOperatorRun() string {
	//This is in assumtion to ENV variables for index image
	cmd := exec.Command("preflight", "check", "operator", "quay.io/opdev/simple-demo-operator-bundle:v0.0.2")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + string(output))
	} else {
		fmt.Printf("output: %s \n\n\n", string(output))

	}
	return string(output)
}
