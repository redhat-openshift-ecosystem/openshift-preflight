// Package artifacts provides functionality for writing artifact files in configured
// artifacts directory. This package operators with a singleton directory variable that can be
// changed and reset. It provides simple functionality that can be accessible from
// any calling library.
package artifacts

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// appFS is the base path FS to base writes on
var appFS = afero.NewOsFs()

// ads is the artifacts directory singleton.
var ads string

// DefaultArtifactsDir is the default value for the directory.
const DefaultArtifactsDir = "artifacts"

func init() {
	Reset()
}

// SetDir sets the package level artifacts directory. This
// can be a relative path or a full path.
func SetDir(s string) {
	fullPath := s
	if !strings.HasPrefix(s, "/") {
		cwd, _ := os.Getwd()
		fullPath = filepath.Join(cwd, s)
	}
	ads = fullPath
}

// Reset restores the default value for the Artifacts Directory.
func Reset() {
	// set the singleton to the default value.
	cwd, _ := os.Getwd()
	ads = filepath.Join(cwd, DefaultArtifactsDir)
}

// WriteFile will write contents of the string to a file in
// the artifacts directory. It will create the artifacts dir
// if necessary.
// Returns the full path (including the artifacts dir)
func WriteFile(filename string, contents io.Reader) (string, error) {
	fullFilePath := filepath.Join(Path(), filename)

	if err := afero.SafeWriteReader(appFS, fullFilePath, contents); err != nil {
		return fullFilePath, fmt.Errorf("could not write file to artifacts directory: %v", err)
	}
	return fullFilePath, nil
}

// Path will return the artifacts directory.
func Path() string {
	return ads
}
