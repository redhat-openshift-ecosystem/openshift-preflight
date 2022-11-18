package artifacts

import (
	"errors"
	"io"
)

var ErrFileAlreadyExists = errors.New("file already exists")

// MapWriter implements an ArtifactWriter storing contents in a map.
type MapWriter struct {
	files map[string]io.Reader
}

// NewMapWriter creates an artifact writer in memory using a map.
func NewMapWriter() (*MapWriter, error) {
	return &MapWriter{
		files: map[string]io.Reader{},
	}, nil
}

// WriteFile places contents into files at filename.
func (w *MapWriter) WriteFile(filename string, contents io.Reader) (string, error) {
	if _, exists := w.files[filename]; exists {
		return "", ErrFileAlreadyExists
	}

	w.files[filename] = contents
	return filename, nil
}

func (w *MapWriter) Files() map[string]io.Reader {
	return w.files
}
