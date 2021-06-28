package errors

import "errors"

var ErrNoChecksEnabled = errors.New("no checks have been enabled")
var ErrRequestedCheckNotFound = errors.New("requested check not found")
var ErrRequestedFormatterNotFound = errors.New("requested formatter is not known")
var ErrFormatterNameNotProvided = errors.New("formatter name is required")
var ErrFormattingResults = errors.New("error formatting results")
var ErrFeatureNotImplemented = errors.New("feature not implemented") // TODO remove this ASAP
var ErrInsufficientPosArguments = errors.New("not enough positional arguments")
var ErrNoResponseFormatSpecified = errors.New("no response format specified")
var ErrGetRemoteContainerFailed = errors.New("failed to pull remote container")
var ErrSaveContainerFailed = errors.New("failed to save container tarball")
var ErrExtractingTarball = errors.New("failed to extract tarball")
var ErrCreateTempDir = errors.New("failed to create temporary directory")
