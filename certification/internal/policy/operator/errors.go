package operator

import "errors"

var (
	ErrK8sAPICallFailed  = errors.New("unable to fetch the requested resource from k8s API server")
	ErrUnsupportedGoType = errors.New("go type unsupported")
)
