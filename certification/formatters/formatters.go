// Package formatters defines the abstractions used to properly format a preflight
// Result.
package formatters

import (
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
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
	Format(runtime.Results) (response []byte, formattingError error)
}

// FormatterFunc describes a function that formats the check validation
// results.
type FormatterFunc = func(runtime.Results) (response []byte, formattingError error)

// NewForConfig returns a new formatter based on the user-provided configuration. It relies
// on config values which should align with known/supported/built-in formatters.
func NewForConfig(cfg runtime.Config) (ResponseFormatter, error) {
	formatter, defined := availableFormatters[cfg.ResponseFormat]
	if !defined {
		return nil, fmt.Errorf(
			"failed to create a new formatter from config \"%s\": %w",
			cfg.ResponseFormat,
			errors.ErrRequestedFormatterNotFound,
		)
	}

	return formatter, nil
}

// New returns a new formatter with the provided name and FormatterFunc.
func New(name, extension string, fn FormatterFunc) (ResponseFormatter, error) {
	if len(name) == 0 {
		return nil, fmt.Errorf(
			"failed to create a new generic formatter: %w",
			errors.ErrFormatterNameNotProvided,
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
	formatterFunc FormatterFunc
}

// Name returns a string identification of the formatter that's in use.
func (f *genericFormatter) PrettyName() string {
	return f.name
}

// Format returns the formatted results as a byte slice.
func (f *genericFormatter) Format(r runtime.Results) ([]byte, error) {
	return f.formatterFunc(r)
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
