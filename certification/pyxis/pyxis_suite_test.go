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
	case strings.Contains(req.URL.Path, "test-results"):
		results = `{"image": "quay.io/awesome/image:latest", "passed": false}`
	case strings.Contains(req.URL.Path, "certification"):
		results = `{"certification_status":"Started","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`

		if req.Method == http.MethodPatch {
			results = `{"certification_status":"In Progress","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`
		}
	case strings.Contains(req.URL.Path, "rpm-manifest"):
		results = `{"object_type": "containerImageRPMManifest"}`
	case strings.Contains(req.URL.Path, "images"):
		results = `{"certified":false,"deleted":false,"image_id":"123456789abc"}`
	}

	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader([]byte(results)))}, nil
}

type fakeHttpCertProjectUnauthorizedClient struct{}

func (fhc fakeHttpCertProjectUnauthorizedClient) Do(req *http.Request) (*http.Response, error) {
	var results string

	switch {
	case strings.Contains(req.URL.Path, "certification"):
		results = `{}`
	}

	return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(bytes.NewReader([]byte(results)))}, nil
}

type fakeHttpCreateImageConflictClient struct{}

func (fhc fakeHttpCreateImageConflictClient) Do(req *http.Request) (*http.Response, error) {
	var results string
	statusCode := http.StatusOK

	switch {
	case strings.Contains(req.URL.Path, "test-results"):
		results = `{"image": "quay.io/awesome/image:latest", "passed": false}`
	case strings.Contains(req.URL.RawQuery, "filter=docker_image_digest=="):
		results = `{"data":[{"certified":false,"deleted":false,"image_id":"123456789abc"}]}`
	case strings.Contains(req.URL.Path, "certification"):
		results = `{"certification_status":"Started","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`

		if req.Method == http.MethodPatch {
			results = `{"certification_status":"In Progress","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`
		}
	case strings.Contains(req.URL.Path, "rpm-manifest"):
		results = `{"object_type": "containerImageRPMManifest"}`
	case strings.Contains(req.URL.Path, "images"):
		results = ``
		statusCode = http.StatusConflict
	}

	return &http.Response{StatusCode: statusCode, Body: io.NopCloser(bytes.NewReader([]byte(results)))}, nil
}

type fakeHttpCreateImageUnauthorizedClient struct{}

func (fhc fakeHttpCreateImageUnauthorizedClient) Do(req *http.Request) (*http.Response, error) {
	var results string
	statusCode := http.StatusOK

	switch {
	case strings.Contains(req.URL.Path, "certification"):
		results = `{"certification_status":"Started","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`

		if req.Method == http.MethodPatch {
			results = `{"certification_status":"In Progress","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`
		}
	case strings.Contains(req.URL.Path, "images"):
		results = `{}`
		statusCode = http.StatusUnauthorized
	}

	return &http.Response{StatusCode: statusCode, Body: io.NopCloser(bytes.NewReader([]byte(results)))}, nil
}

type fakeHttpCreateImageConflictAndUnauthorizedClient struct{}

func (fhc fakeHttpCreateImageConflictAndUnauthorizedClient) Do(req *http.Request) (*http.Response, error) {
	var results string
	statusCode := http.StatusOK

	switch {
	case strings.Contains(req.URL.Path, "test-results"):
		results = `{"image": "quay.io/awesome/image:latest", "passed": false}`
	case strings.Contains(req.URL.RawQuery, "filter=docker_image_digest=="):
		results = ``
		statusCode = http.StatusUnauthorized
	case strings.Contains(req.URL.Path, "certification"):
		results = `{"certification_status":"Started","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`

		if req.Method == http.MethodPatch {
			results = `{"certification_status":"In Progress","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`
		}
	case strings.Contains(req.URL.Path, "rpm-manifest"):
		results = `{"object_type": "containerImageRPMManifest"}`
	case strings.Contains(req.URL.Path, "images"):
		results = ``
		statusCode = http.StatusConflict
	}

	return &http.Response{StatusCode: statusCode, Body: io.NopCloser(bytes.NewReader([]byte(results)))}, nil
}

type fakeHttpCreateRPMManifestConflictClient struct{}

func (fhc fakeHttpCreateRPMManifestConflictClient) Do(req *http.Request) (*http.Response, error) {
	var results string
	statusCode := http.StatusOK

	switch {
	case strings.Contains(req.URL.Path, "test-results"):
		results = `{"image": "quay.io/awesome/image:latest", "passed": false}`
	case strings.Contains(req.URL.Path, "certification"):
		results = `{"certification_status":"Started","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`

		if req.Method == http.MethodPatch {
			results = `{"certification_status":"In Progress","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`
		}
	case strings.Contains(req.URL.Path, "rpm-manifest"):
		if req.Method == http.MethodPost {
			results = `{}`
			statusCode = http.StatusConflict
		}

		if req.Method == http.MethodGet {
			results = `{"object_type": "containerImageRPMManifest"}`
		}
	case strings.Contains(req.URL.Path, "images"):
		results = `{"certified":false,"deleted":false,"image_id":"123456789abc"}`

	}

	return &http.Response{StatusCode: statusCode, Body: io.NopCloser(bytes.NewReader([]byte(results)))}, nil
}

type fakeHttpCreateRPMManifestConflictAndUnauthorizedClient struct{}

func (fhc fakeHttpCreateRPMManifestConflictAndUnauthorizedClient) Do(req *http.Request) (*http.Response, error) {
	var results string
	statusCode := http.StatusOK

	switch {
	case strings.Contains(req.URL.Path, "test-results"):
		results = `{"image": "quay.io/awesome/image:latest", "passed": false}`
	case strings.Contains(req.URL.Path, "certification"):
		results = `{"certification_status":"Started","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`

		if req.Method == http.MethodPatch {
			results = `{"certification_status":"In Progress","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`
		}
	case strings.Contains(req.URL.Path, "rpm-manifest"):
		if req.Method == http.MethodPost {
			results = `{}`
			statusCode = http.StatusConflict
		}

		if req.Method == http.MethodGet {
			results = `{}`
			statusCode = http.StatusUnauthorized
		}
	case strings.Contains(req.URL.Path, "images"):
		results = `{"certified":false,"deleted":false,"image_id":"123456789abc"}`
	}

	return &http.Response{StatusCode: statusCode, Body: io.NopCloser(bytes.NewReader([]byte(results)))}, nil
}

type fakeHttpCreateRPMManifestUnauthorizedClient struct{}

func (fhc fakeHttpCreateRPMManifestUnauthorizedClient) Do(req *http.Request) (*http.Response, error) {
	var results string
	statusCode := http.StatusOK

	switch {
	case strings.Contains(req.URL.Path, "test-results"):
		results = `{"image": "quay.io/awesome/image:latest", "passed": false}`
	case strings.Contains(req.URL.Path, "certification"):
		results = `{"certification_status":"Started","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`

		if req.Method == http.MethodPatch {
			results = `{"certification_status":"In Progress","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`
		}
	case strings.Contains(req.URL.Path, "rpm-manifest"):
		results = `{}`
		statusCode = http.StatusUnauthorized
	case strings.Contains(req.URL.Path, "images"):
		results = `{"certified":false,"deleted":false,"image_id":"123456789abc"}`
	}

	return &http.Response{StatusCode: statusCode, Body: io.NopCloser(bytes.NewReader([]byte(results)))}, nil
}

type fakeHttpCreateTestResultsUnauthorizedClient struct{}

func (fhc fakeHttpCreateTestResultsUnauthorizedClient) Do(req *http.Request) (*http.Response, error) {
	var results string
	statusCode := http.StatusOK

	switch {
	case strings.Contains(req.URL.Path, "test-results"):
		results = `{}`
		statusCode = http.StatusUnauthorized
	case strings.Contains(req.URL.Path, "certification"):
		results = `{"certification_status":"Started","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`

		if req.Method == http.MethodPatch {
			results = `{"certification_status":"In Progress","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`
		}
	case strings.Contains(req.URL.Path, "rpm-manifest"):
		results = `{"object_type": "containerImageRPMManifest"}`
	case strings.Contains(req.URL.Path, "images"):
		results = `{"certified":false,"deleted":false,"image_id":"123456789abc"}`
	}

	return &http.Response{StatusCode: statusCode, Body: io.NopCloser(bytes.NewReader([]byte(results)))}, nil
}
