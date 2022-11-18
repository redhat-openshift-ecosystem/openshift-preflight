// Package formatters defines the abstractions used to properly format a preflight
// Result.
package formatters

import (
	"context"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/config"
)

// ResponseFormatter describes the expected methods a formatter
// must implement.
type ResponseFormatter interface {
	// PrettyName is the name used to represent this formatter.
	PrettyName() string
	// FileExtension represents the file extension one might use when creating
	// a file with the contents of this formatter.
	FileExtension() string
	// Format takes Results, formats it as needed, and returns the formatted
	// results ready to write as a byte slice.
	Format(context.Context, certification.Results) (response []byte, formattingError error)
}

// NewForConfig returns a new formatter based on the user-provided configuration. It relies
// on config values which should align with known/supported/built-in formatters.
func NewForConfig(cfg config.Config) (ResponseFormatter, error) {
	return NewByName(cfg.ResponseFormat())
}

// NewByName returns a predefined ResponseFormatter with the given name.
// TODO: New* funcs in this package may benefit from renaming.
func NewByName(name string) (ResponseFormatter, error) {
	formatter, defined := availableFormatters[name]
	if !defined {
		return nil, fmt.Errorf("%s: %s",
			"The requested formatter is unknown",
			name,
		)
	}

	return formatter, nil
}

// New returns a new formatter with the provided name and FormatterFunc.
func New(name, extension string, fn formatters.FormatterFunc) (ResponseFormatter, error) {
	if len(name) == 0 {
		return nil, fmt.Errorf(
			"failed to create a new generic formatter: formatter name is required",
		)
	}

	gf := genericFormatter{
		name:          name,
		formatterFunc: fn,
		fileExtension: extension,
	}

	return &gf, nil
}

// genericFormatter represents a generic approach to formatting that implements the
// ResponseFormatter interface. Can be leveraged to build a custom formatter quickly.
type genericFormatter struct {
	name          string
	fileExtension string
	formatterFunc formatters.FormatterFunc
}

// Name returns a string identification of the formatter that's in use.
func (f *genericFormatter) PrettyName() string {
	return f.name
}

// Format returns the formatted results as a byte slice.
func (f *genericFormatter) Format(ctx context.Context, r certification.Results) ([]byte, error) {
	return f.formatterFunc(ctx, r)
}

// FileExtension returns the extension a user might use when formatting
// results with this formatter and writing that to disk.
func (f *genericFormatter) FileExtension() string {
	return f.fileExtension
}

// availableFormatters maps configuration-friendly values to pretty representations
// of the same value, and their corresponding Formatter included with this library.
var availableFormatters = map[string]ResponseFormatter{
	"json":     &genericFormatter{"Generic JSON", "json", genericJSONFormatter},
	"xml":      &genericFormatter{"Generic XML", "xml", genericXMLFormatter},
	"junitxml": &genericFormatter{"JUnit XML", "xml", junitXMLFormatter},
}
