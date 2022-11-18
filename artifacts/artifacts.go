// Package artifacts provides functionality for writing artifact files in configured
// artifacts directory. This package operators with a singleton directory variable that can be
// changed and reset. It provides simple functionality that can be accessible from
// any calling library.
package artifacts

import (
	"context"
	"io"
)

const DefaultArtifactsDir = "artifacts"

// ContextWithWriter adds ArtifactWriter w to the context ctx.
func ContextWithWriter(ctx context.Context, w ArtifactWriter) context.Context {
	return context.WithValue(ctx, artifactWriterContextKey, w)
}

// WriterFromContext returns the writer from the context, or nil.
func WriterFromContext(ctx context.Context) ArtifactWriter {
	w := ctx.Value(artifactWriterContextKey)
	if writer, ok := w.(ArtifactWriter); ok {
		return writer
	}

	return nil
}

// contextKey is a key used to store/retrieve ArtifactsWriter in/from context.Context.
type contextKey string

const artifactWriterContextKey contextKey = "ArtifactWriter"

// ArtifactWriter is the functionality required by all implementations.
type ArtifactWriter interface {
	WriteFile(filename string, contents io.Reader) (fullpathToFile string, err error)
}
