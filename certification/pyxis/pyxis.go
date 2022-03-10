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
	req, err := p.newRequestWithApiToken(ctx, http.MethodPost, getPyxisUrl("images"), bytes.NewReader(b))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	log.Debugf("URL is: %s", req.URL)

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

	if resp.StatusCode == 409 {
		return nil, errors.Err409StatusCode
	}

	if !checkStatus(resp.StatusCode) {
		log.Errorf("%s: %s", "received non 200 status code in createImage", string(body))
		return nil, errors.ErrNon200StatusCode
	}

	var newCertImage CertImage
	if err := json.Unmarshal(body, &newCertImage); err != nil {
		log.Error(err)
		return nil, err
	}

	return &newCertImage, nil
}

func (p *pyxisEngine) getImage(ctx context.Context, dockerImageDigest string) (*CertImage, error) {
	req, err := p.newRequestWithApiToken(ctx, http.MethodGet,
		getPyxisUrl(fmt.Sprintf("projects/certification/id/%s/images?filter=docker_image_digest==%s", p.ProjectId, dockerImageDigest)), nil)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	log.Debugf("URL is: %s", req.URL)

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
		log.Errorf("%s: %s", "received non 200 status code in getImage", string(body))
		return nil, errors.ErrNon200StatusCode
	}

	// using an inline struct since this api's response is in a different format
	data := struct {
		Data []CertImage `json:"data,omitempty"`
	}{}

	if err := json.Unmarshal(body, &data); err != nil {
		log.Error(err)
		return nil, err
	}

	return &data.Data[0], nil
}

func (p *pyxisEngine) createRPMManifest(ctx context.Context, rpmManifest *RPMManifest) (*RPMManifest, error) {
	b, err := json.Marshal(rpmManifest)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	req, err := p.newRequestWithApiToken(ctx, http.MethodPost, getPyxisUrl(fmt.Sprintf("images/id/%s/rpm-manifest", rpmManifest.ImageID)), bytes.NewReader(b))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	log.Debugf("URL is: %s", req.URL)

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

	if resp.StatusCode == 409 {
		return nil, errors.Err409StatusCode
	}

	if !checkStatus(resp.StatusCode) {
		log.Errorf("%s: %s", "received non 200 status code in createRPMManifest", string(body))
		return nil, errors.ErrNon200StatusCode
	}

	var newRPMManifest RPMManifest
	if err := json.Unmarshal(body, &newRPMManifest); err != nil {
		log.Error(err)
		return nil, err
	}

	return &newRPMManifest, nil
}

func (p *pyxisEngine) getRPMManifest(ctx context.Context, imageID string) (*RPMManifest, error) {
	req, err := p.newRequestWithApiToken(ctx, http.MethodGet, getPyxisUrl(fmt.Sprintf("images/id/%s/rpm-manifest", imageID)), nil)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	log.Debugf("URL is: %s", req.URL)

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
		log.Errorf("%s: %s", "received non 200 status code in getRPMManifest", string(body))
		return nil, errors.ErrNon200StatusCode
	}

	var newRPMManifest RPMManifest
	if err := json.Unmarshal(body, &newRPMManifest); err != nil {
		log.Error(err)
		return nil, err
	}

	return &newRPMManifest, nil
}

func (p *pyxisEngine) GetProject(ctx context.Context) (*CertProject, error) {
	req, err := p.newRequestWithApiToken(ctx, http.MethodGet, getPyxisUrl(fmt.Sprintf("projects/certification/id/%s", p.ProjectId)), nil)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	log.Debugf("URL is: %s", req.URL)

	resp, err := p.Client.Do(req)
	if err != nil {
		log.Error(err, "client.Do failed")
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err, "readall failed")
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
	req, err := p.newRequestWithApiToken(ctx, http.MethodPatch, getPyxisUrl(fmt.Sprintf("projects/certification/id/%s", p.ProjectId)), bytes.NewReader(b))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	log.Debugf("URL is: %s", req.URL)

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
	req, err := p.newRequestWithApiToken(ctx, http.MethodPost, getPyxisUrl(fmt.Sprintf("projects/certification/id/%s/test-results", p.ProjectId)), bytes.NewReader(b))
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
		log.Errorf("%s: %s", "received non 200 status code in createTestResults", string(body))
		return nil, errors.ErrNon200StatusCode
	}

	newTestResults := TestResults{}
	if err := json.Unmarshal(body, &newTestResults); err != nil {
		log.Error(err)
		return nil, err
	}

	return &newTestResults, nil
}

func (p *pyxisEngine) newRequestWithApiToken(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-API-KEY", p.ApiToken)

	if body != nil {
		req.Header.Add("Content-type", "application/json")
	}

	return req, nil
}

// checkStatus is used to check for a 2xx status code
func checkStatus(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}
