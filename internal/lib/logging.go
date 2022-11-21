package lib

import (
	"bytes"
	"context"
	"io"

	"github.com/sirupsen/logrus"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"
)

// LogThroughArtifactWriterIfSet reconfigures the logger used by Preflight to write to
// the artifact writer if one is configured. If this is called and no Artifact Writer
// is configured, this will set writes to the application's logger to be discarded.
func LogThroughArtifactWriterIfSet(ctx context.Context) {
	if w := artifacts.WriterFromContext(ctx); w != nil {
		log.L().SetLevel(logrus.TraceLevel)
		log.L().SetFormatter(&logrus.TextFormatter{})
		b := bytes.NewBufferString("")
		log.L().SetOutput(b)

		w.WriteFile("preflight.log", b) //nolint:errcheck
		return
	}

	log.L().SetOutput(io.Discard)
}

type contextKey string

var executionEnvIsCLI = contextKey("IsCLI")

func CallerIsCLI(ctx context.Context) bool {
	val := ctx.Value(executionEnvIsCLI)
	switch b := val.(type) {
	case bool:
		return b
	default:
		return false
	}
}

// SetCallerToCLI sets the caller as the CLI. NOTE: This is a temporary
// workaround for internal CLI executions and will be removed at a later
// date.
func SetCallerToCLI(ctx context.Context) context.Context {
	return context.WithValue(ctx, executionEnvIsCLI, true)
}
