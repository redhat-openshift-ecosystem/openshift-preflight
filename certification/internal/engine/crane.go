package engine

import (
	"github.com/google/go-containerregistry/pkg/crane"
)

type craneEngine struct{}

func NewCraneEngine() *craneEngine {
	return &craneEngine{}
}

func (c *craneEngine) ListTags(imageURI string) ([]string, error) {
	return crane.ListTags(imageURI)
}
