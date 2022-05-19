package container

import "errors"

var (
	ErrLicensesNotADir        = errors.New("licenses is not a directory")
	ErrDetectingModifiedFiles = errors.New("error detecting modified files")
	ErrExtractingLayer        = errors.New("could not extract layer")
)
