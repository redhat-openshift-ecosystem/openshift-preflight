package pyxis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/shurcooL/graphql"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"
)

const (
	apiVersion = "v1"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type pyxisClient struct {
	APIToken  string
	ProjectID string
	Client    HTTPClient
	PyxisHost string
}

func (p *pyxisClient) getPyxisURL(path string) string {
	return fmt.Sprintf("https://%s/%s/%s", p.PyxisHost, apiVersion, path)
}

func (p *pyxisClient) getPyxisGraphqlURL() string {
	return fmt.Sprintf("https://%s/graphql/", p.PyxisHost)
}

func NewPyxisClient(pyxisHost string, apiToken string, projectID string, httpClient HTTPClient) *pyxisClient {
	return &pyxisClient{
		APIToken:  apiToken,
		ProjectID: projectID,
		Client:    httpClient,
		PyxisHost: pyxisHost,
	}
}

func (p *pyxisClient) createImage(ctx context.Context, certImage *CertImage) (*CertImage, error) {
	logger := logr.FromContextOrDiscard(ctx)
	b, err := json.Marshal(certImage)
	if err != nil {
		return nil, fmt.Errorf("could not marshal certImage: %w", err)
	}
	req, err := p.newRequestWithAPIToken(ctx, http.MethodPost, p.getPyxisURL("images"), bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	logger.V(log.TRC).Info("pyxis URL", "url", req.URL)

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot create image in pyxis: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read body: %w", err)
	}

	if resp.StatusCode == http.StatusConflict {
		return nil, ErrPyxis409StatusCode
	}

	if ok := checkStatus(resp.StatusCode); !ok {
		return nil, fmt.Errorf(
			"status code: %d: body: %s",
			resp.StatusCode,
			string(body))
	}

	var newCertImage CertImage
	if err := json.Unmarshal(body, &newCertImage); err != nil {
		return nil, fmt.Errorf("could not unmarshal body: %s: %w", string(body), err)
	}

	return &newCertImage, nil
}

func (p *pyxisClient) getImage(ctx context.Context, dockerImageDigest string) (*CertImage, error) {
	logger := logr.FromContextOrDiscard(ctx)
	req, err := p.newRequestWithAPIToken(ctx, http.MethodGet,
		p.getPyxisURL(fmt.Sprintf("projects/certification/id/%s/images?filter=docker_image_digest==%s", p.ProjectID, dockerImageDigest)), nil)
	if err != nil {
		return nil, fmt.Errorf("could not create new request: %w", err)
	}

	logger.V(log.TRC).Info("pyxis URL", "url", req.URL)

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get image from pyxis: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read body: %w", err)
	}

	if ok := checkStatus(resp.StatusCode); !ok {
		return nil, fmt.Errorf(
			"status code: %d: body: %s",
			resp.StatusCode,
			string(body))
	}

	// using an inline struct since this api's response is in a different format
	data := struct {
		Data []CertImage `json:"data"`
	}{}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("could not unmarshal body: %s: %w", string(body), err)
	}

	return &data.Data[0], nil
}

// updateImage updates a given certification image based on how the image is built in the `submit` flow
func (p *pyxisClient) updateImage(ctx context.Context, certImage *CertImage) (*CertImage, error) {
	// instantiating a patchCertImage struct, so we only send the minimum fields required to pyxis
	patchCertImage := &CertImage{
		ID:           certImage.ID,
		Architecture: certImage.Architecture,
		Certified:    certImage.Certified,
	}

	b, err := json.Marshal(patchCertImage)
	if err != nil {
		return nil, fmt.Errorf("could not marshal certImage: %w", err)
	}
	req, err := p.newRequestWithAPIToken(ctx, http.MethodPatch, p.getPyxisURL(fmt.Sprintf("images/id/%s", patchCertImage.ID)), bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot update image in pyxis: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	if ok := checkStatus(resp.StatusCode); !ok {
		return nil, fmt.Errorf(
			"status code: %d: body: %s",
			resp.StatusCode,
			string(body))
	}

	var updatedCertImage CertImage
	if err := json.Unmarshal(body, &updatedCertImage); err != nil {
		return nil, fmt.Errorf("could not unmarshal body: %s: %w", string(body), err)
	}

	return &updatedCertImage, nil
}

// FindImagesByDigest uses an unauthenticated call to find_images() graphql function, and will
// return a slice of CertImages. It accepts a slice of image digests. The query return is then
// packed into the slice of CertImages.
func (p *pyxisClient) FindImagesByDigest(ctx context.Context, digests []string) ([]CertImage, error) {
	if len(digests) == 0 {
		return nil, fmt.Errorf("no digests specified")
	}
	// our graphQL query
	var query struct {
		FindImages struct {
			// Additional fields for return should be added here
			ContainerImage []struct {
				ID                graphql.String  `graphql:"_id"`
				Certified         graphql.Boolean `graphql:"certified"`
				DockerImageDigest graphql.String  `graphql:"docker_image_digest"`
			} `graphql:"data"`
			Error struct {
				Status graphql.Int    `graphql:"status"`
				Detail graphql.String `graphql:"detail"`
			} `graphql:"error"`
			Total graphql.Int
			Page  graphql.Int
			// filter to make sure we get exact results
		} `graphql:"find_images(filter: {docker_image_digest:{in:$digests}})"`
	}

	graphqlDigests := make([]graphql.String, len(digests))
	for idx, digest := range digests {
		graphqlDigests[idx] = graphql.String(digest)
	}
	// variables to feed to our graphql filter
	variables := map[string]interface{}{
		"digests": graphqlDigests,
	}

	// make our query
	httpClient, ok := p.Client.(*http.Client)
	if !ok {
		return nil, fmt.Errorf("client could not be used as http.Client")
	}
	client := graphql.NewClient(p.getPyxisGraphqlURL(), httpClient)

	err := client.Query(ctx, &query, variables)
	if err != nil {
		return nil, fmt.Errorf("error while executing find_images query: %v", err)
	}

	images := make([]CertImage, len(query.FindImages.ContainerImage))
	for idx, image := range query.FindImages.ContainerImage {
		images[idx] = CertImage{
			ID:                string(image.ID),
			Certified:         bool(image.Certified),
			DockerImageDigest: string(image.DockerImageDigest),
		}
	}

	return images, nil
}

func (p *pyxisClient) createRPMManifest(ctx context.Context, rpmManifest *RPMManifest) (*RPMManifest, error) {
	logger := logr.FromContextOrDiscard(ctx)

	b, err := json.Marshal(rpmManifest)
	if err != nil {
		return nil, fmt.Errorf("could not marshal rpm manifest: %w", err)
	}
	req, err := p.newRequestWithAPIToken(ctx, http.MethodPost, p.getPyxisURL(fmt.Sprintf("images/id/%s/rpm-manifest", rpmManifest.ImageID)), bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("could not create new request: %w", err)
	}

	logger.V(log.TRC).Info("pyxis URL", "url", req.URL)

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not create rpm manifest in pyxis: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read body: %w", err)
	}

	if resp.StatusCode == 409 {
		return nil, ErrPyxis409StatusCode
	}

	if ok := checkStatus(resp.StatusCode); !ok {
		return nil, fmt.Errorf(
			"status code: %d: body: %s",
			resp.StatusCode,
			string(body))
	}

	var newRPMManifest RPMManifest
	if err := json.Unmarshal(body, &newRPMManifest); err != nil {
		return nil, fmt.Errorf("could not unmarshal body: %s: %w", string(body), err)
	}

	return &newRPMManifest, nil
}

func (p *pyxisClient) getRPMManifest(ctx context.Context, imageID string) (*RPMManifest, error) {
	logger := logr.FromContextOrDiscard(ctx)

	req, err := p.newRequestWithAPIToken(ctx, http.MethodGet, p.getPyxisURL(fmt.Sprintf("images/id/%s/rpm-manifest", imageID)), nil)
	if err != nil {
		return nil, fmt.Errorf("could not create new request: %w", err)
	}

	logger.V(log.TRC).Info("pyxis URL", "url", req.URL)

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get rpm manifest from pyxis: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read body: %w", err)
	}

	if ok := checkStatus(resp.StatusCode); !ok {
		return nil, fmt.Errorf(
			"status code: %d: body: %s",
			resp.StatusCode,
			string(body))
	}

	var newRPMManifest RPMManifest
	if err := json.Unmarshal(body, &newRPMManifest); err != nil {
		return nil, fmt.Errorf("could not unmarshal body: %s: %w", string(body), err)
	}

	return &newRPMManifest, nil
}

func (p *pyxisClient) GetProject(ctx context.Context) (*CertProject, error) {
	logger := logr.FromContextOrDiscard(ctx)

	req, err := p.newRequestWithAPIToken(ctx, http.MethodGet, p.getPyxisURL(fmt.Sprintf("projects/certification/id/%s", p.ProjectID)), nil)
	if err != nil {
		return nil, fmt.Errorf("could not create new request: %v", err)
	}

	logger.V(log.TRC).Info("pyxis URL", "url", req.URL)

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get project from pyxis: %v", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read body: %v", err)
	}

	if ok := checkStatus(resp.StatusCode); !ok {
		return nil, fmt.Errorf(
			"status code: %d: body: %s",
			resp.StatusCode,
			string(body))
	}

	var certProject CertProject
	if err := json.Unmarshal(body, &certProject); err != nil {
		return nil, fmt.Errorf("could not unmarshal body: %s: %v", string(body), err)
	}

	return &certProject, nil
}

func (p *pyxisClient) updateProject(ctx context.Context, certProject *CertProject) (*CertProject, error) {
	logger := logr.FromContextOrDiscard(ctx)

	// We cannot send the project type or container type
	// to pyxis in a Patch. Copy the CertProject and strip type
	// values to have omitempty skip the key in the JSON patch.
	patchCertProject := &CertProject{
		ID:                  certProject.ID,
		CertificationStatus: certProject.CertificationStatus,
		Container:           certProject.Container,
		Name:                certProject.Name,
		ProjectStatus:       certProject.ProjectStatus,
		// Do not copy the Type.
	}
	patchCertProject.Container.Type = "" // Truncate this value, too.

	b, err := json.Marshal(patchCertProject)
	if err != nil {
		return nil, fmt.Errorf("could not marshal certProject: %w", err)
	}
	req, err := p.newRequestWithAPIToken(ctx, http.MethodPatch, p.getPyxisURL(fmt.Sprintf("projects/certification/id/%s", p.ProjectID)), bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("could not create new request: %w", err)
	}

	logger.V(log.TRC).Info("pyxis URL", "url", req.URL)

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not update project in pyxis: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read body: %w", err)
	}

	if ok := checkStatus(resp.StatusCode); !ok {
		return nil, fmt.Errorf(
			"status code: %d: body: %s",
			resp.StatusCode,
			string(body))
	}

	var newCertProject CertProject
	if err := json.Unmarshal(body, &newCertProject); err != nil {
		return nil, fmt.Errorf("could not unmarshal body: %s: %w", string(body), err)
	}

	return &newCertProject, nil
}

func (p *pyxisClient) createTestResults(ctx context.Context, testResults *TestResults) (*TestResults, error) {
	b, err := json.Marshal(testResults)
	if err != nil {
		return nil, fmt.Errorf("could not marshal test results: %w", err)
	}
	req, err := p.newRequestWithAPIToken(ctx, http.MethodPost, p.getPyxisURL(fmt.Sprintf("projects/certification/id/%s/test-results", p.ProjectID)), bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("could not create new request: %w", err)
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not create test results in pyxis: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read body: %w", err)
	}

	if ok := checkStatus(resp.StatusCode); !ok {
		return nil, fmt.Errorf(
			"status code: %d: body: %s",
			resp.StatusCode,
			string(body))
	}

	newTestResults := TestResults{}
	if err := json.Unmarshal(body, &newTestResults); err != nil {
		return nil, fmt.Errorf("could not unmarshal body: %s: %w", string(body), err)
	}

	return &newTestResults, nil
}

func (p *pyxisClient) createArtifact(ctx context.Context, artifact *Artifact) (*Artifact, error) {
	logger := logr.FromContextOrDiscard(ctx)

	b, err := json.Marshal(artifact)
	if err != nil {
		return nil, fmt.Errorf("could not marshal artifact: %w", err)
	}
	req, err := p.newRequestWithAPIToken(ctx, http.MethodPost, p.getPyxisURL(fmt.Sprintf("projects/certification/id/%s/artifacts", p.ProjectID)), bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("could not create new request: %w", err)
	}

	logger.V(log.TRC).Info("pyxis URL", "url", req.URL)

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not create artifact in pyxis: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read body: %w", err)
	}

	if ok := checkStatus(resp.StatusCode); !ok {
		return nil, fmt.Errorf(
			"status code: %d: body: %s",
			resp.StatusCode,
			string(body))
	}

	var newArtifact Artifact
	if err := json.Unmarshal(body, &newArtifact); err != nil {
		return nil, fmt.Errorf("could not unmarshal body: %s: %w", string(body), err)
	}

	return &newArtifact, nil
}

func (p *pyxisClient) newRequestWithAPIToken(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error) {
	req, err := p.newRequest(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-API-KEY", p.APIToken)

	return req, nil
}

func (p *pyxisClient) newRequest(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Add("Content-type", "application/json")
	}

	return req, nil
}

// checkStatus is used to check for a 2xx status code
func checkStatus(statusCode int) bool {
	return statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices
}
