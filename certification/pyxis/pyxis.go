package pyxis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	var newCertImage CertImage
	if err := json.Unmarshal(body, &newCertImage); err != nil {
		log.Error(err)
		return nil, err
	}

	return &newCertImage, nil
}

func (p *pyxisEngine) createRPMManifest(ctx context.Context, imageId string, rpms []RPM) error {
	rpmManifest := RPMManifest{
		ImageID: imageId,
		RPMS:    rpms,
	}
	b, err := json.Marshal(rpmManifest)
	if err != nil {
		log.Error(err)
		return err
	}
	req, err := newRequestWithApiToken(ctx, http.MethodPost, getPyxisUrl(fmt.Sprintf("images/id/%s/rpm-manifest", imageId)), p.ApiToken)
	if err != nil {
		log.Error(err)
		return err
	}
	req = withJson(req, b)

	_, err = p.Client.Do(req)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
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

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return nil, err
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

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return nil, err
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

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return nil, err
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
