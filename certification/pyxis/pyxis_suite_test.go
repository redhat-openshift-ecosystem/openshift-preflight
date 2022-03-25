package pyxis

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func TestPyxis(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pyxis Engine Suite")
}

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.TraceLevel)
	viper.SetEnvPrefix("pflt")
	viper.AutomaticEnv()
}

type localRoundTripper struct {
	handler http.Handler
}

func (l localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	l.handler.ServeHTTP(w, req)
	return w.Result(), nil
}

func mustWrite(w io.Writer, s string) {
	_, err := io.WriteString(w, s)
	if err != nil {
		panic(err)
	}
}

type (
	pyxisProjectHandler     struct{}
	pyxisImageHandler       struct{}
	pyxisRPMManifestHandler struct{}
	pyxisTestResultsHandler struct{}
)

func (p *pyxisProjectHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	log.Trace("In the Project ServeHTTP")
	response.Header().Set("Content-Type", "application/json")
	if request.Body != nil {
		defer request.Body.Close()
	}
	switch {
	case request.Header["X-Api-Key"][0] == "my-bad-project-api-token":
		response.WriteHeader(401)
	case request.Method == http.MethodPost:
		body, err := io.ReadAll(request.Body)
		if err != nil {
			response.WriteHeader(400)
		}
		mustWrite(response, string(body))
	default:
		mustWrite(response, `{"_id":"deadb33f","certification_status":"Started","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`)
	}
	return
}

func (p *pyxisImageHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	log.Trace("In the Image ServeHTTP")
	response.Header().Set("Content-Type", "application/json")
	if request.Body != nil {
		defer request.Body.Close()
	}
	switch {
	case strings.Contains(request.URL.Path, "my-image-409-project-id") && request.Method == http.MethodPost:
		response.WriteHeader(409)
	case request.Header["X-Api-Key"][0] == "my-bad-image-api-token":
		response.WriteHeader(401)
	default:
		mustWrite(response, `{"_id":"blah","certified":false,"deleted":false,"image_id":"123456789abc"}`)
	}
	return
}

func (p *pyxisRPMManifestHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	log.Trace("In the RPM Manifest ServeHTTP")
	response.Header().Set("Content-Type", "application/json")
	if request.Body != nil {
		defer request.Body.Close()
	}
	switch {
	case strings.Contains(request.URL.Path, "my-manifest-409-project-id") && request.Method == http.MethodPost:
		response.WriteHeader(409)
		mustWrite(response, `{"_id":"foo"}`)
	case request.Header["X-Api-Key"][0] == "my-bad-rpmmanifest-api-token":
		response.WriteHeader(401)
	default:
		mustWrite(response, `{"_id":"blah"}`)
	}
	return
}

func (p *pyxisTestResultsHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	log.Trace("In the Results ServeHTTP")
	response.Header().Set("Content-Type", "application/json")
	if request.Body != nil {
		defer request.Body.Close()
	}
	switch {
	case request.Header["X-Api-Key"][0] == "my-bad-testresults-api-token":
		response.WriteHeader(401)
	default:
		mustWrite(response, `{"image":"quay.io/awesome/image:latest","passed": true}`)
	}
	return
}
