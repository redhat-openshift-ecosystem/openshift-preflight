package certification

import v1 "github.com/google/go-containerregistry/pkg/v1"

// ImasgeReference holds all things image-related
type ImageReference struct {
	ImageURI        string
	ImageFSPath     string
	ImageInfo       v1.Image
	ImageRepository string
	ImageRegistry   string
	ImageTagOrSha   string
}
