package operatorsdk

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"

	"github.com/sirupsen/logrus"
)

func TestOperatorSdk(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operator Suite")
}

func init() {
	log.L().SetFormatter(&logrus.TextFormatter{})
	log.L().SetLevel(logrus.TraceLevel)
}
