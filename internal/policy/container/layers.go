package container

import (
	"fmt"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
)

// getImageLayers returns the root filesystem DiffIDs of the image.
func getImageLayers(image cranev1.Image) ([]cranev1.Hash, error) {
	if image == nil {
		return nil, fmt.Errorf("image information is required")
	}

	configFile, err := image.ConfigFile()
	if err != nil {
		//coverage:ignore
		return nil, err
	}

	return configFile.RootFS.DiffIDs, nil
}
