package container

import (
	"context"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/pyxis"
)

// fakePyxisClient implements lib.PyxisClient for testing
type fakePyxisClient struct {
	getProjectFunc func(ctx context.Context) (*pyxis.CertProject, error)
}

func (f *fakePyxisClient) GetProject(ctx context.Context) (*pyxis.CertProject, error) {
	if f.getProjectFunc != nil {
		return f.getProjectFunc(ctx)
	}
	return nil, nil
}

func (f *fakePyxisClient) FindImagesByDigest(ctx context.Context, digests []string) ([]pyxis.CertImage, error) {
	return nil, nil
}

func (f *fakePyxisClient) SubmitResults(ctx context.Context, certInput *pyxis.CertificationInput) (*pyxis.CertificationResults, error) {
	return nil, nil
}

// Helper functions to return different project types
func returnBundleProject(ctx context.Context) (*pyxis.CertProject, error) {
	return &pyxis.CertProject{
		Name: "test-bundle-project",
		Container: pyxis.Container{
			Type: "operator bundle image",
		},
	}, nil
}

func returnContainerProject(ctx context.Context) (*pyxis.CertProject, error) {
	return &pyxis.CertProject{
		Name: "test-container-project",
		Container: pyxis.Container{
			Type: "container",
		},
	}, nil
}
