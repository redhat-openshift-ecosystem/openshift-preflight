package formatters

import "errors"

var (
	ErrRequestedFormatterNotFound = errors.New("requested formatter is not known")
	ErrFormatterNameNotProvided   = errors.New("formatter name is required")
	ErrFormattingResults          = errors.New("error formatting results")
)
