package csv

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCSVLib(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CSV Suite")
}
