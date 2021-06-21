package formatters

import (
	"fmt"

	"github.com/komish/preflight/certification/errors"
	"github.com/komish/preflight/certification/runtime"
)

func junitXMLFormatter(r runtime.Results) ([]byte, error) {
	return nil, fmt.Errorf("%w: The JUnit XML Formatter is not implemented", errors.ErrFeatureNotImplemented)
}
