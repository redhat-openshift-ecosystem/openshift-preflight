package runtime

import (
	"io"
	"os"
)

// ResultWriterFile implements a ResultWriter for use at preflight runtime.
type ResultWriterFile struct {
	file *os.File
}

// OpenFile will open the expected file for writing.
func (f *ResultWriterFile) OpenFile(name string) (io.WriteCloser, error) {
	file, err := os.OpenFile(
		name,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o600)
	if err != nil {
		return nil, err
	}

	f.file = file // so we can close it later.
	return f, nil
}

func (f *ResultWriterFile) Close() error {
	return f.file.Close()
}

func (f *ResultWriterFile) Write(p []byte) (int, error) {
	return f.file.Write(p)
}
