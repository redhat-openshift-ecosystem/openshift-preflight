package container

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestContainerLib(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "lib container suite")
}
