// Package artifacts provides functionality for writing artifact files in configured
// artifacts directory. This package operators with a singleton directory variable that can be
// changed and reset. It provides simple functionality that can be accessible from
// any calling library.
package artifacts

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ads is the artifacts directory singleton.
var ads string

// DefaultArtifactsDir is the default value for the directory.
const DefaultArtifactsDir = "artifacts"

func init() {
	// set the singleton to the default value.
	ads = DefaultArtifactsDir
}

// SetDir sets the package level artifacts directory. This
// can be a relative path or a full path.
func SetDir(s string) {
	ads = s
}

// Reset restores the default value for the Artifacts Directory.
func Reset() {
	ads = DefaultArtifactsDir
}

// WriteFile will write contents of the string to a file in
// the artifacts directory. It will create the artifacts dir
// if necessary.
// Returns the full path (including the artifacts dir)
func WriteFile(filename, contents string) (string, error) {
	artifactDir := ads
	artifactDir, err := createArtifactsDir(artifactDir)
	if err != nil {
		// Fatal does an os.Exit. If we can't create the artifacts directory,
		// we can't continue.
		log.Fatal(fmt.Errorf("could not create artifact path: %v", err))
	}
	fullFilePath := filepath.Join(Path(), filename)

	if err := os.WriteFile(fullFilePath, []byte(contents), 0o644); err != nil {
		return fullFilePath, fmt.Errorf("could not write file to artifacts diretory: %v", err)
	}
	return fullFilePath, nil
}

// createArtifactsDir creates the artifacts directory at path artifactsDir.
// If the path is not a full path, this will resolve the full path.
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

// Path will return the artifacts directory.
func Path() string {
	return ads
}
