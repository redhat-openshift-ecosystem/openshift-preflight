package engine

import (
	"context"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/authn"
)

type craneEngine struct{}

func NewCraneEngine() *craneEngine {
	return &craneEngine{}
}

func (c *craneEngine) ListTags(ctx context.Context, imageURI string) ([]string, error) {
	options := []crane.Option{
		crane.WithContext(ctx),
		crane.WithAuthFromKeychain(authn.PreflightKeychain),
	}

	return crane.ListTags(imageURI, options...)
}
