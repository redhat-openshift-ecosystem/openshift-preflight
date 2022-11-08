package errors

import "errors"

// Library-wide error messages are here.
var (
	ErrKubeconfigEmpty              = errors.New("kubeconfig value is empty")
	ErrIndexImageEmpty              = errors.New("index image value is empty")
	ErrImageEmpty                   = errors.New("image is empty")
	ErrCannotResolvePolicyException = errors.New("cannot resolve policy exception")
	ErrCannotInitializeChecks       = errors.New("unable to initialize checks")
)
