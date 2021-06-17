package errors

import "errors"

var ErrNoPoliciesEnabled = errors.New("no policies have been enabled")
var ErrRequestedPolicyNotFound = errors.New("requested policy not found")
var ErrFormattingResults = errors.New("error formatting results")
var ErrFeatureNotImplemented = errors.New("feature not implemented") // TODO remove this ASAP
var ErrInsufficientPosArguments = errors.New("not enough positional arguments")
