package pyxis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	apiVersion = "v1"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type pyxisEngine struct {
	ApiToken  string
	ProjectId string
	Client    HTTPClient
}

func getPyxisUrl(path string) string {
	return fmt.Sprintf("https://%s/%s/%s", viper.GetString("pyxis_host"), apiVersion, path)
}

func NewPyxisEngine(apiToken string, projectId string, httpClient HTTPClient) *pyxisEngine {
	return &pyxisEngine{
		ApiToken:  apiToken,
		ProjectId: projectId,
		Client:    httpClient,
	}
}

func (p *pyxisEngine) createImage(ctx context.Context, certImage *CertImage) (*CertImage, error) {
	b, err := json.Marshal(certImage)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	req, err := newRequestWithApiToken(ctx, http.MethodPost, getPyxisUrl("images"), p.ApiToken)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	req = withJson(req, b)

	resp, err := p.Client.Do(req)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if !checkStatus(resp.StatusCode) {
		log.Errorf("%s: %s", "received non 200 status code in create manifest call", string(body))
		return nil, errors.ErrNon200StatusCode
	}

	var newCertImage CertImage
	if err := json.Unmarshal(body, &newCertImage); err != nil {
		log.Error(err)
		return nil, err
	}

	return &newCertImage, nil
}

func (p *pyxisEngine) createRPMManifest(ctx context.Context, rpmManifest *RPMManifest) (*RPMManifest, error) {
	b, err := json.Marshal(rpmManifest)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	req, err := newRequestWithApiToken(ctx, http.MethodPost, getPyxisUrl(fmt.Sprintf("images/id/%s/rpm-manifest", rpmManifest.ImageID)), p.ApiToken)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	req = withJson(req, b)

	resp, err := p.Client.Do(req)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	var newRPMManifest RPMManifest
	if err := json.Unmarshal(body, &newRPMManifest); err != nil {
		log.Error(err)
		return nil, err
	}

	if !checkStatus(resp.StatusCode) {
		log.Errorf("%s: %s", "received non 200 status code in createRPMManifest", string(body))
		return nil, errors.ErrNon200StatusCode
	}

	return &newRPMManifest, nil
}

func (p *pyxisEngine) GetProject(ctx context.Context) (*CertProject, error) {
	req, err := newRequestWithApiToken(ctx, http.MethodGet, getPyxisUrl(fmt.Sprintf("projects/certification/id/%s", p.ProjectId)), p.ApiToken)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if !checkStatus(resp.StatusCode) {
		log.Errorf("%s: %s", "received non 200 status code in GetProject", string(body))
		return nil, errors.ErrNon200StatusCode
	}

	var certProject CertProject
	if err := json.Unmarshal(body, &certProject); err != nil {
		log.Error(err)
		return nil, err
	}

	return &certProject, nil
}

func (p *pyxisEngine) updateProject(ctx context.Context, certProject *CertProject) (*CertProject, error) {
	b, err := json.Marshal(certProject)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	req, err := newRequestWithApiToken(ctx, http.MethodPatch, getPyxisUrl(fmt.Sprintf("projects/certification/id/%s", p.ProjectId)), p.ApiToken)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	req = withJson(req, b)

	resp, err := p.Client.Do(req)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if !checkStatus(resp.StatusCode) {
		log.Errorf("%s: %s", "received non 200 status code in updateProject", string(body))
		return nil, errors.ErrNon200StatusCode
	}

	var newCertProject CertProject
	if err := json.Unmarshal(body, &newCertProject); err != nil {
		log.Error(err)
		return nil, err
	}

	return &newCertProject, nil
}

func (p *pyxisEngine) createTestResults(ctx context.Context, testResults *TestResults) (*TestResults, error) {
	b, err := json.Marshal(testResults)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	req, err := newRequestWithApiToken(ctx, http.MethodPost, getPyxisUrl(fmt.Sprintf("projects/certification/id/%s/test-results", p.ProjectId)), p.ApiToken)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	req = withJson(req, b)

	resp, err := p.Client.Do(req)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if !checkStatus(resp.StatusCode) {
		log.Errorf("%s: %s", "received non 200 status code in createTestResults", string(body))
		return nil, errors.ErrNon200StatusCode
	}

	var newTestResults = TestResults{}
	if err := json.Unmarshal(body, &newTestResults); err != nil {
		log.Error(err)
		return nil, err
	}

	return &newTestResults, nil
}

func newRequestWithApiToken(ctx context.Context, method string, url string, token string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-API-TOKEN", token)

	return req, nil
}

func withJson(req *http.Request, b []byte) *http.Request {
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Content-length", fmt.Sprint(len(b)))
	req.Body = io.NopCloser(bytes.NewReader(b))
	return req
}

// checkStatus is used to check for a 2xx status code
func checkStatus(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}
