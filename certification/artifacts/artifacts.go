package artifacts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// WriteFile will write contents of the string to a file in
// the artifacts directory.
// Returns the full path (including the artifacts dir)
func WriteFile(filename, contents string) (string, error) {
	fullFilePath := filepath.Join(Path(), filename)

	err := os.WriteFile(fullFilePath, []byte(contents), 0o644)
	if err != nil {
		return fullFilePath, fmt.Errorf("could not write file to artifacts diretory: %v", err)
	}
	return fullFilePath, nil
}

func createArtifactsDir(artifactsDir string) (string, error) {
	if !strings.HasPrefix(artifactsDir, "/") {
		currentDir, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("unable to get current directory: %w", err)
		}

		artifactsDir = filepath.Join(currentDir, artifactsDir)
	}

	err := os.MkdirAll(artifactsDir, 0o777)
	if err != nil {
		return "", fmt.Errorf("unable to create artifactsDir: %w", err)
	}
	return artifactsDir, nil
}

// Path will return the artifacts path from viper config
func Path() string {
	artifactDir := viper.GetString("artifacts")
	artifactDir, err := createArtifactsDir(artifactDir)
	if err != nil {
		// Fatal does an os.Exit. If we can't create the artifacts directory,
		// we can't continue.
		log.Fatal(fmt.Errorf("could not retrieve artifact path: %v", err))
	}
	return filepath.Join(artifactDir)
}
