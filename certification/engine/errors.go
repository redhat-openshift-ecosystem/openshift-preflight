package engine

import "errors"

var (
	ErrNoChecksEnabled        = errors.New("no checks have been enabled")
	ErrRequestedCheckNotFound = errors.New("requested check not found")
)
