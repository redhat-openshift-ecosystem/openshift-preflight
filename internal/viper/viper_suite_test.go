package viper

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestViper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Package-Local Viper")
}
