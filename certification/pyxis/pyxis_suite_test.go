package pyxis

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2/dsl/core"
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
	pyxisProjectHandler           struct{}
	pyxisImageHandler             struct{}
	pyxisRPMManifestHandler       struct{}
	pyxisTestResultsHandler       struct{}
	pyxisGraphqlLayerHandler      struct{}
	pyxisGraphqlFindImagesHandler struct{}
	errorHandler                  struct{}
)

// For each of these ServeHTTP methods, there is a main switch statement that controls what response will
// be sent back. Most are switched on the API Key that is provided. Depending on the request.Method, it
// will determine whether to send back an error, or a valid response. This can probably be improved a bit
// to dedupe. Acknowledged that it is a bit fragile. -bpc

func (p *pyxisProjectHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	log.Trace("In the Project ServeHTTP")
	response.Header().Set("Content-Type", "application/json")
	if request.Body != nil {
		defer request.Body.Close()
	}
	switch {
	case request.Method == http.MethodGet && request.Header["X-Api-Key"][0] == "my-401-project-api-token":
		response.WriteHeader(http.StatusUnauthorized)
	case request.Header["X-Api-Key"][0] == "my-bad-project-api-token":
		response.WriteHeader(http.StatusUnauthorized)
	case request.Header["X-Api-Key"][0] == "my-index-docker-io-project-api-token":
		mustWrite(response, `{"_id":"deadb33f","certification_status":"Started","name":"My Index Docker IO Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers","registry":"docker.io", "repository":"my/repo"}}`)
	case request.Method == http.MethodPatch && request.Header["X-Api-Key"][0] == "my-error-project-api-token":
		response.WriteHeader(http.StatusInternalServerError)
	case request.Method == http.MethodPost:
		body, err := io.ReadAll(request.Body)
		if err != nil {
			response.WriteHeader(http.StatusBadRequest)
		}
		mustWrite(response, string(body))
	default:
		mustWrite(response, `{"_id":"deadb33f","certification_status":"Started","name":"My Spiffy Project","project_status":"Foo","type":"Containers","container":{"docker_config_json":"{}","type":"Containers"}}`)
	}
}

func (p *pyxisImageHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	log.Trace("In the Image ServeHTTP")
	response.Header().Set("Content-Type", "application/json")
	if request.Body != nil {
		defer request.Body.Close()
	}
	responseString := `{"_id":"blah","certified":false,"deleted":false,"image_id":"123456789abc"}`
	log.Tracef("Method: %s", request.Method)
	switch {
	case request.Method == http.MethodPost && request.Header["X-Api-Key"][0] == "my-image-409-api-token":
		response.WriteHeader(http.StatusConflict)
	case request.Method == http.MethodPost && request.Header["X-Api-Key"][0] == "my-bad-401-image-api-token":
		response.WriteHeader(http.StatusConflict)
	case request.Method == http.MethodPost && request.Header["X-Api-Key"][0] == "my-bad-image-api-token":
		response.WriteHeader(http.StatusConflict)
	case request.Method == http.MethodGet && request.Header["X-Api-Key"][0] == "my-bad-401-image-api-token":
		log.Trace("get with 401")
		response.WriteHeader(http.StatusUnauthorized)
	case request.Header["X-Api-Key"][0] == "my-bad-image-api-token":
		response.WriteHeader(http.StatusUnauthorized)
	case request.Method == http.MethodPost && request.Header["X-Api-Key"][0] == "my-bad-500-image-api-token":
		response.WriteHeader(http.StatusInternalServerError)
	case request.Header["X-Api-Key"][0] == "my-index-docker-io-project-api-token":
		mustWrite(response, `{"_id": "blah", "architecture": "amd64", "object_type": "containerImage", "repositories": [ {"published": false, "registry": "docker.io", "repository": "my/repo", "tags": [{"name": "docker_io_v3"}]}]}`)
	case request.Method == http.MethodPost:
		mustWrite(response, responseString)
	default:
		mustWrite(response, fmt.Sprintf(`{"data":[%s]}`, responseString))
	}
}

func (p *pyxisRPMManifestHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	log.Trace("In the RPM Manifest ServeHTTP")
	response.Header().Set("Content-Type", "application/json")
	if request.Body != nil {
		defer request.Body.Close()
	}
	switch {
	case request.Method == http.MethodPost && request.Header["X-Api-Key"][0] == "my-bad-rpmmanifest-409-api-token":
		response.WriteHeader(http.StatusConflict)
		mustWrite(response, `{"_id":"foo"}`)
	case request.Method == http.MethodPost && request.Header["X-Api-Key"][0] == "my-bad-rpmmanifest-401-api-token":
		response.WriteHeader(http.StatusConflict)
	case request.Method == http.MethodGet && request.Header["X-Api-Key"][0] == "my-bad-rpmmanifest-401-api-token":
		response.WriteHeader(http.StatusUnauthorized)
	case request.Header["X-Api-Key"][0] == "my-bad-rpmmanifest-api-token":
		response.WriteHeader(http.StatusUnauthorized)
	default:
		mustWrite(response, `{"_id":"blah"}`)
	}
}

func (p *pyxisTestResultsHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	log.Trace("In the Results ServeHTTP")
	response.Header().Set("Content-Type", "application/json")
	if request.Body != nil {
		defer request.Body.Close()
	}
	switch {
	case request.Header["X-Api-Key"][0] == "my-bad-testresults-api-token":
		response.WriteHeader(http.StatusUnauthorized)
	default:
		mustWrite(response, `{"image":"quay.io/awesome/image:latest","passed": true}`)
	}
}

func (p *pyxisGraphqlLayerHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	log.Trace("In the graphql Layers ServeHTTP")
	response.Header().Set("Content-Type", "application/json")
	if request.Body != nil {
		defer request.Body.Close()
	}
	mustWrite(response, `{
		"data":{
			"find_images":{
				"error":null,
				"total":1,
				"page":0,
				"data":[
					{
						"uncompressed_top_layer_id":"good_top_layer",
						"_id":"deadb33f",
						"freshness_grades":[
							{
								"grade": "A",
								"start_date": "2022-05-03T08:52:00+00:00",
								"end_date": null
							}
						]
					}
				]
			}
		}
	}`)
}

func (p *pyxisGraphqlFindImagesHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	log.Trace("In the graphql FindImages ServeHTTP")
	response.Header().Set("Content-Type", "application/json")
	if request.Body != nil {
		defer request.Body.Close()
	}
	mustWrite(response, `{"data":{"find_images":{"error":null,"total":1,"page":0,"data":[{"_id":"deadb33f","certified":true}]}}}`)
}

func (p *errorHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/json")
	if request.Body != nil {
		defer request.Body.Close()
	}
	response.WriteHeader(http.StatusBadGateway)
}

// In order to test some negative paths, this io.Reader will just throw an error
type errReader int

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("test error")
}
