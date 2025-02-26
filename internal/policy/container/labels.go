package container

import cranev1 "github.com/google/go-containerregistry/pkg/v1"

// getContainerLabels is a helper function to obtain the labels from an images configfile
func getContainerLabels(image cranev1.Image) (map[string]string, error) {
	configFile, err := image.ConfigFile()
	return configFile.Config.Labels, err
}
