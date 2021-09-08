package utils

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func createArtifactsDir(artifactsDir string) (string, error) {
	if !strings.HasPrefix(artifactsDir, "/") {
		currentDir, err := os.Getwd()
		if err != nil {
			log.Error("unable to get current directory: ", err)
			return "", err
		}

		artifactsDir = filepath.Join(currentDir, artifactsDir)
	}

	err := os.MkdirAll(artifactsDir, 0777)
	if err != nil {
		log.Error("unable to create artifactsDir: ", err)
		return "", err
	}
	return artifactsDir, nil
}

func ArtifactPath() string {
	artifactDir := viper.GetString("artifacts")
	artifactDir, err := createArtifactsDir(artifactDir)
	if err != nil {
		log.Fatal("could not retrieve artifact path")
		// Fatal does an os.Exit
	}
	return filepath.Join(artifactDir)
}
