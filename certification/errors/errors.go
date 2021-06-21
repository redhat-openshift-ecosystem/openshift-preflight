package errors

import "errors"

var ErrNoPoliciesEnabled = errors.New("no policies have been enabled")
var ErrRequestedPolicyNotFound = errors.New("requested policy not found")
var ErrRequestedFormatterNotFound = errors.New("requested formatter is not known")
var ErrFormatterNameNotProvided = errors.New("formatter name is required")
var ErrFormattingResults = errors.New("error formatting results")
var ErrFeatureNotImplemented = errors.New("feature not implemented") // TODO remove this ASAP
var ErrInsufficientPosArguments = errors.New("not enough positional arguments")
var ErrNoResponseFormatSpecified = errors.New("no response format specified")
