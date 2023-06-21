package operator

import (
	"context"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/operatorsdk"
)

type operatorSdk interface {
	Scorecard(context.Context, string, operatorsdk.OperatorSdkScorecardOptions) (*operatorsdk.OperatorSdkScorecardReport, error)
}
