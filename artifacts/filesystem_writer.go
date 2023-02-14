package artifacts

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/spf13/afero"
)

// FilesystemWriter is an ArtifactWriter that targets a particular directory on
// the underlying filesystem.
type FilesystemWriter struct {
	dir string
	fs  afero.Fs
}

// NewFilesystemWriter creates an artifact writer which writes to the filesystem.
func NewFilesystemWriter(opts ...FilesystemWriterOption) (*FilesystemWriter, error) {
	w := FilesystemWriter{
		dir: resolveFullPath(DefaultArtifactsDir),
		fs:  afero.NewOsFs(),
	}

	for _, opt := range opts {
		opt(&w)
	}

	return &w, nil
}

// WithDirectory sets the artifacts directory to dir unless it's empty, in which case
// this option is ignored.
func WithDirectory(dir string) FilesystemWriterOption {
	return func(w *FilesystemWriter) {
		if dir == "" {
			return
		}
		w.dir = resolveFullPath(dir)
	}
}

type FilesystemWriterOption = func(*FilesystemWriter)

// WriteFile places contents into dir at filename.
func (w *FilesystemWriter) WriteFile(filename string, contents io.Reader) (string, error) {
	fullFilePath := filepath.Join(w.Path(), filename)

	if err := afero.WriteReader(w.fs, fullFilePath, contents); err != nil {
		return fullFilePath, fmt.Errorf("could not write file to artifacts directory: %v", err)
	}
	return fullFilePath, nil
}

// Exists checks if a file exists with a filename
func (w *FilesystemWriter) Exists(filename string) (bool, error) {
	fullFilePath := filepath.Join(w.Path(), filename)

	return afero.Exists(w.fs, fullFilePath)
}

// Remove removes contents from dir at filename.
func (w *FilesystemWriter) Remove(filename string) error {
	fullFilePath := filepath.Join(w.Path(), filename)

	return w.fs.Remove(fullFilePath)
}

// Path is the full artifacts path.
func (w *FilesystemWriter) Path() string {
	return w.dir
}
