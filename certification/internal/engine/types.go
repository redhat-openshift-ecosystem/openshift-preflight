package engine

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type ImageReference struct {
	ImageURI    string
	ImageFSPath string
	ImageInfo   *v1.Image
}

type RegistryCredentials struct {
	Username string
	Password string
}
