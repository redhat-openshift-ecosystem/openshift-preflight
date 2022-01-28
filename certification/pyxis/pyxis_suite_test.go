package pyxis

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

func TestPyxis(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pyxis Engine Suite")
}

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.TraceLevel)
}

type fakeHttpClient struct{}

func (fhc fakeHttpClient) Do(req *http.Request) (*http.Response, error) {
	var results string

	switch {
	case strings.Contains(req.URL.Path, "certification"):
		results = `{"certification_status":"Started","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`

		if req.Method == http.MethodPatch {
			results = `{"certification_status":"In Progress","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`
		}
	case strings.Contains(req.URL.Path, "images"):
		results = `{"certified":false,"deleted":false}`
	case strings.Contains(req.URL.Path, "test-results"):
		results = `{"image": "quay.io/awesome/image:latest", "passed": false,}`
	}

	return &http.Response{Body: io.NopCloser(bytes.NewReader([]byte(results)))}, nil
}
