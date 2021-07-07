package cmd

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCMD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CMD Suite")
}
