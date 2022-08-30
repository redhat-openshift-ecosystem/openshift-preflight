package operator

import (
	"context"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/bundle"

	log "github.com/sirupsen/logrus"
)

var _ certification.Check = &securityContextConstraintsInCSV{}

// securityContextConstraintsInCSV evaluates the csv and logs a message if a non default security context constraint is
// needed by the operator
type securityContextConstraintsInCSV struct {
	customSCCSpecified bool
}

func NewSecurityContextConstraintsCheck() *securityContextConstraintsInCSV {
	return &securityContextConstraintsInCSV{
		false,
	}
}

func (p *securityContextConstraintsInCSV) Validate(ctx context.Context, bundleRef certification.ImageReference) (bool, error) {
	requestedSCCList, err := p.dataToValidate(ctx, bundleRef.ImageFSPath)
	if err != nil {
		return false, err
	}

	return p.validate(ctx, requestedSCCList)
}

func (p *securityContextConstraintsInCSV) dataToValidate(ctx context.Context, imagePath string) ([]string, error) {
	requestedSccList, err := bundle.GetSecurityContextConstraints(ctx, imagePath)
	if err != nil {
		return nil, fmt.Errorf("unable to extract security context constraints from ClusterServiceVersion: %w", err)
	}

	return requestedSccList, nil
}

//nolint:unparam // ctx is unused. Keep for future use.
func (p *securityContextConstraintsInCSV) validate(ctx context.Context, requestedSccList []string) (bool, error) {
	if len(requestedSccList) == 0 {
		log.Infof("No custom security context constraint was detected in the CSV. The default restricted SCC will be used.")
		return true, nil
	}

	if len(requestedSccList) != 1 {
		return false, fmt.Errorf("only one scc should be requested at a time")
	}

	log.Infof("A custom scc was specified: %s , action may be needed by a cluster admin, please check the "+
		"operator documentation for more information", requestedSccList[0])
	// set bool to change help text
	p.customSCCSpecified = true
	return true, nil
}

func (p *securityContextConstraintsInCSV) Name() string {
	return "SecurityContextConstraintsInCSV"
}

func (p *securityContextConstraintsInCSV) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Evaluates the csv and logs a message if a non default security context constraint is needed by the operator",
		Level:            "optional",
		KnowledgeBaseURL: "https://redhat-connect.gitbook.io/certified-operator-guide/troubleshooting-and-resources/sccs", // Placeholder
		CheckURL:         "https://redhat-connect.gitbook.io/certified-operator-guide/troubleshooting-and-resources/sccs", // Placeholder
	}
}

func (p *securityContextConstraintsInCSV) Help() certification.HelpText {
	// TODO implement a way to display help text for informational checks
	// Currently this bool triggered customized help is not shown anywhere since this check is purely informational
	// and always passes. Since the ask is to indicate this information to the user it should eventually be shown in the
	// output rather than just logs
	if p.customSCCSpecified {
		return certification.HelpText{
			Message: "A custom security context constraint was detected in the csv. " +
				"Please see the operators documentation for details.",
			Suggestion: "A custom security context constraint may need to be applied by a cluster admin.",
		}
	}
	// This is the default help text returned in case the check fails
	return certification.HelpText{
		Message: "The securityContextConstraintsInCSV logs a message if the operator requests a " +
			"security context constraint. Please review the operators documentation to see if this is needed.",
		Suggestion: "If no scc is detected the default restricted scc will be used.",
	}
}
