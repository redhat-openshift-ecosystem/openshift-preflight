package shell

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	// apiEndpoint is the endpoint used to query for package uniqueness.
	apiEndpoint = "https://catalog.redhat.com/api/containers/v1/operators/packages"

	// packageKey is the packageKey in annotations.yaml that contains the package name.
	packageKey = "operators.operatorframework.io.bundle.package.v1"
)

// apiRespondData is the response received from the defined API
type apiResponseData struct {
	Data     []packageData `json:"data"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	Total    int           `json:"total"`
}

// packageData represents a single package entry in the API response.
type packageData struct {
	Id             string `json:"_id"`
	Association    string `json:"association"`
	CreateDate     string `json:"creation_date"`
	LastUpdateTime string `json:"last_update_date"`
	PackageName    string `json:"package_name"`
	Source         string `json:"source"`
}

// OperatorPkgNameIsUniqueMountedCheck finds the package name as defined in the operator bundle's annotations
// and checks it against Red Hat APIs to confirm that the package name is unique at the time of the
// check.
type OperatorPkgNameIsUniqueMountedCheck struct{}

func (p *OperatorPkgNameIsUniqueMountedCheck) Validate(bundle string) (bool, error) {
	annotations, err := p.getAnnotationsFromBundle(bundle)
	if err != nil {
		log.Errorf("unable to get annotations.yaml from the bundle")
		return false, err
	}

	packageName, err := p.getPackageName(annotations)
	if err != nil {
		log.Error("unable to extract package name from ClusterServicVersion", err)
		return false, err
	}

	log.Debugf("operator package name is %s", packageName)

	req, err := p.buildRequest(apiEndpoint, packageName)
	if err != nil {
		log.Error("unable to build API request structure", err)
		return false, err
	}

	resp, err := p.queryAPI(http.DefaultClient, req)
	if err != nil {
		log.Error("unable to query package name validation API for uniqueness check", err)
		return false, err
	}

	data, err := p.parseAPIResponse(resp)
	if err != nil {
		log.Error("unable to parse response provided by package name validation API", err)
		return false, err
	}

	return p.validate(data)
}

// getAnnotationsFromBundle will search a bundle's metadata directory for a file called annotations.yaml, which is
// expected to exist in every bundle. This file's contents are read and marshaled to a simple map, which is then
// returned.
func (p *OperatorPkgNameIsUniqueMountedCheck) getAnnotationsFromBundle(mountedDir string) (map[string]string, error) {
	log.Trace("reading annotations file from the bundle")
	log.Debug("mounted directory is ", mountedDir)
	annotationsFilePath := path.Join(mountedDir, "metadata", "annotations.yaml")

	fileContents, err := os.ReadFile(annotationsFilePath)
	if err != nil {
		log.Error("fail to read metadata/annotation.yaml file in bundle")
		return nil, err
	}

	annotations, err := p.extractAnnotationsBytes(fileContents)
	if err != nil {
		log.Error("metadata/annotations.yaml found but is malformed")
		return nil, err
	}

	return annotations, nil
}

// extractAnnotationsBytes reads the annotation data read from a file and returns the expected format for that yaml
// represented as a map[string]string.
func (p *OperatorPkgNameIsUniqueMountedCheck) extractAnnotationsBytes(annotationBytes []byte) (map[string]string, error) {
	if len(annotationBytes) == 0 {
		return nil, errors.New("the annotations file was empty")
	}

	var bundleMeta metadata
	if err := yaml.Unmarshal(annotationBytes, &bundleMeta); err != nil {
		log.Error("metadata/annotations.yaml found but is malformed")
		return nil, err
	}

	return bundleMeta.Annotations, nil
}

// getPackageName accepts the annotations map and searches for the specified annotation corresponding
// with the complete bundle name for an operator, which is then returned.
func (p *OperatorPkgNameIsUniqueMountedCheck) getPackageName(annotations map[string]string) (string, error) {
	log.Tracef("searching for package key (%s) in bundle", packageKey)
	log.Trace("bundle data: ", annotations)
	pkg, found := annotations[packageKey]
	if !found {
		return "", fmt.Errorf("did not find package name at the key %s in the annotations.yaml", packageKey)
	}

	return pkg, nil
}

// buildRequest builds the http.Request using the input parameters and returns a client.
func (p *OperatorPkgNameIsUniqueMountedCheck) buildRequest(apiURL, packageName string) (*http.Request, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// this endpoint supports a query string, and we use that to determine if a
	// package with the name already exists.
	queryString := req.URL.Query()
	queryString.Add("filter", fmt.Sprintf("package_name==%s", packageName))
	req.URL.RawQuery = queryString.Encode()

	return req, nil
}

// queryAPI uses the provided client to query the remote API, and returns the response if it
// response is successful, or an error if the response was unexpected in any way.
func (p *OperatorPkgNameIsUniqueMountedCheck) queryAPI(client apiClient, request *http.Request) (*http.Response, error) {
	log.Trace("making API request to ", request.URL.String())
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	log.Trace("response code: ", resp.Status)

	// The Connect API returns a 200 regardless of whether the package was found or not. Until this
	// assumption changes, we assume any non-200 response is invalid, or due to a malformed request.
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("received an unexpected status code for the request")
	}

	return resp, nil
}

// parseAPIResponse reads the response and checks the body for the expected contents, and then
// returns the body content as apiResponseData.
func (p *OperatorPkgNameIsUniqueMountedCheck) parseAPIResponse(resp *http.Response) (*apiResponseData, error) {
	var data apiResponseData
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Trace("response body: ", string(body))

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

// validate checks the apiResponseData and confirms that the package is unique by confirming that the
// API returned no packages using the same name.
func (p *OperatorPkgNameIsUniqueMountedCheck) validate(resp *apiResponseData) (bool, error) {
	// success case - the API returned no entries
	if len(resp.Data) == 0 {
		return true, nil
	}

	log.Error("a package already exists in the Red Hat ecosystem using the same name")
	// we don't expect multiple entries, but resp.Data is a list so we will iterate.
	for _, v := range resp.Data {
		log.Error("found the following entry: ", v)
	}

	return false, nil
}

func (p *OperatorPkgNameIsUniqueMountedCheck) Name() string {
	return "OperatorPackageNameIsUniqueMounted"
}

func (p *OperatorPkgNameIsUniqueMountedCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Validating Bundle image package name uniqueness",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *OperatorPkgNameIsUniqueMountedCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check encountered an error. It is possible that the bundle package name already exist in the RedHat Catalog registry.",
		Suggestion: "Bundle package name must be unique meaning that it doesn't already exist in the RedHat Catalog registry",
	}
}

// apiClient is a simple interface encompassing the only http.Client method we utilize for preflight checks. This exists to
// enable mock implementations for testing purposes.
type apiClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type metadata struct {
	Annotations map[string]string
}
