package formatters

import (
	"context"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
)

// FormatterFunc describes a function that formats the check validation
// results.
type FormatterFunc = func(context.Context, certification.Results) (response []byte, formattingError error)
