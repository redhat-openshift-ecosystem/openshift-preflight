package operator

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/operator-framework/api/pkg/manifests"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/bundle"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	libcsv "github.com/redhat-openshift-ecosystem/openshift-preflight/internal/csv"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"
)

var infraAnnotations = map[string]string{
	libcsv.DisconnectedAnnotation:   "required",
	libcsv.FIPSCompliantAnnotation:  "required",
	libcsv.ProxyAwareAnnotation:     "required",
	libcsv.TLSProfilesAnnotation:    "required",
	libcsv.TokenAuthAWSAnnotation:   "required",
	libcsv.TokenAuthAzureAnnotation: "required",
	libcsv.TokenAuthGCPAnnotation:   "required",
	libcsv.CNFAnnotation:            "optional",
	libcsv.CNIAnnotation:            "optional",
	libcsv.CSIAnnotation:            "optional",
}

var _ check.Check = &RequiredAnnotations{}

type RequiredAnnotations struct{}

func (h RequiredAnnotations) Validate(ctx context.Context, imageReference image.ImageReference) (result bool, err error) {
	return h.validate(ctx, imageReference.ImageFSPath)
}

func (h RequiredAnnotations) getBundleCSV(_ context.Context, bundlepath string) (*operatorsv1alpha1.ClusterServiceVersion, error) {
	bundle, err := manifests.GetBundleFromDir(bundlepath)
	if err != nil {
		return nil, err
	}
	return bundle.CSV, nil
}

func (h RequiredAnnotations) validate(ctx context.Context, bundledir string) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)
	csv, err := h.getBundleCSV(ctx, bundledir)
	if err != nil {
		return false, err
	}

	var missingAnnotations []string
	incorrectValues := map[string]string{}
	for annotation, status := range infraAnnotations {
		value, ok := csv.GetAnnotations()[annotation]
		if !ok {
			// only add the required annotations to the missing list
			if status == "required" {
				missingAnnotations = append(missingAnnotations, annotation)
			}
			continue
		}
		// the only string values allowed are lower case 'true' or 'false'
		// nolint:staticcheck // the conditional is more readable with negation on the outside
		if !(value == "true" || value == "false") {
			incorrectValues[annotation] = value
		}
	}

	if len(missingAnnotations) > 0 {
		logger.V(log.DBG).Info("expected annotations are missing", "missingAnnotations", missingAnnotations)
	}

	if len(incorrectValues) > 0 {
		for key, value := range incorrectValues {
			logger.V(log.DBG).Info(fmt.Sprintf("expected annotation: %s to have either 'true' or 'false' value, but had value of: %s.", key, value))
		}
	}

	return len(missingAnnotations) == 0 && len(incorrectValues) == 0, nil
}

func (h RequiredAnnotations) Name() string {
	return "RequiredAnnotations"
}

func (h RequiredAnnotations) Metadata() check.Metadata {
	return check.Metadata{
		Description:      "Checks that the CSV has all of the required feature annotations.",
		Level:            check.LevelBest,
		KnowledgeBaseURL: "https://access.redhat.com/documentation/en-us/red_hat_software_certification/2024/html-single/red_hat_openshift_software_certification_policy_guide/index#con-operator-requirements_openshift-sw-cert-policy-products-managed",
		CheckURL:         "https://access.redhat.com/documentation/en-us/red_hat_software_certification/2024/html-single/red_hat_openshift_software_certification_policy_guide/index#con-operator-requirements_openshift-sw-cert-policy-products-managed",
	}
}

func (h RequiredAnnotations) Help() check.HelpText {
	return check.HelpText{
		Message:    "Check that the CSV has all of the required feature annotations.",
		Suggestion: "Add all of the required annotations, and make sure the value is set to either 'true' or 'false'",
	}
}

func (h RequiredAnnotations) RequiredFilePatterns() []string {
	return bundle.BundleFiles
}
