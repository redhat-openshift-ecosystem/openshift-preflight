package formatters

import (
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
)

// ResponseFormatter describes the expected methods a formatter
// must implement.
type ResponseFormatter interface {
	PrettyName() string
	Format(runtime.Results) (response []byte, formattingError error)
}

// GenericFormatter represents a generic approach to formatting that implements the
// ResponseFormatter interface. Can be leveraged to build a custom formatter quickly.
type GenericFormatter struct {
	Name          string
	FormatterFunc FormatterFunc
}

// Name returns a string identification of the formatter that's in use.
func (f *GenericFormatter) PrettyName() string {
	return f.Name
}

func (f *GenericFormatter) Format(r runtime.Results) ([]byte, error) {
	return f.FormatterFunc(r)
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
func New(name string, fn FormatterFunc) (ResponseFormatter, error) {
	if len(name) == 0 {
		return nil, fmt.Errorf(
			"failed to create a new generic formatter: %w",
			errors.ErrFormatterNameNotProvided,
		)
	}

	gf := GenericFormatter{
		Name:          name,
		FormatterFunc: fn,
	}

	return &gf, nil
}

// availableFormatters maps configuration-friendly values to pretty representations
// of the same value, and their corresponding Formatter included with this library.
var availableFormatters = map[string]ResponseFormatter{
	"json":     &GenericFormatter{"Generic JSON", genericJSONFormatter},
	"xml":      &GenericFormatter{"Generic XML", genericXMLFormatter},
	"junitxml": &GenericFormatter{"JUnit XML", junitXMLFormatter},
}

// AllFormats returns all formats and formatters made available by this library.
func AllFormats() []string {
	all := make([]string, len(availableFormatters))
	i := 0

	for k := range availableFormatters {
		all[i] = k
		i++
	}

	return all
}
