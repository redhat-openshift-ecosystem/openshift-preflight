package operator

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/api/pkg/manifests"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/bundle"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	libcsv "github.com/redhat-openshift-ecosystem/openshift-preflight/internal/csv"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
)

var _ check.Check = &FollowsRestrictedNetworkEnablementGuidelines{}

type FollowsRestrictedNetworkEnablementGuidelines struct{}

func (p FollowsRestrictedNetworkEnablementGuidelines) Validate(ctx context.Context, imgRef image.ImageReference) (bool, error) {
	return p.validate(ctx, imgRef.ImageFSPath)
}

//nolint:unparam // ctx is unused. Keep for future use.
func (p FollowsRestrictedNetworkEnablementGuidelines) getBundleCSV(ctx context.Context, bundlepath string) (*operatorsv1alpha1.ClusterServiceVersion, error) {
	bundle, err := manifests.GetBundleFromDir(bundlepath)
	if err != nil {
		return nil, err
	}
	return bundle.CSV, nil
}

func (p FollowsRestrictedNetworkEnablementGuidelines) validate(ctx context.Context, bundledir string) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)
	csv, err := p.getBundleCSV(ctx, bundledir)
	if err != nil {
		return false, err
	}

	// in this case, it is more readable to declare the variable first and have subsequent conditional assignments
	// nolint:staticcheck
	restrictedNetworkSupport := false

	// If the CSV does not claim to support disconnected environments, there's no reason to check that it followed guidelines.
	if libcsv.HasDisconnectedAnnotation(csv) && libcsv.SupportsDisconnected(csv.Annotations[libcsv.DisconnectedAnnotation]) {
		restrictedNetworkSupport = true
	}

	if libcsv.HasInfrastructureFeaturesAnnotation(csv) && libcsv.SupportsDisconnectedViaInfrastructureFeatures(csv.Annotations[libcsv.InfrastructureFeaturesAnnotation]) {
		restrictedNetworkSupport = true
	}

	if !restrictedNetworkSupport {
		logger.Info("this operator does not indicate it supports installation into restricted networks. This is safe to ignore if you are not intending to deploy in these environments.")
		return false, nil
	}

	// You must have at least one related image (your controller manager) in order to be considered restricted-network ready
	if !libcsv.HasRelatedImages(csv) {
		logger.Info("this operator did not have any related images, and at least one is expected")
		return false, nil
	}

	// All related images must be pinned. No tag references.
	if !libcsv.RelatedImagesArePinned(csv.Spec.RelatedImages) {
		logger.Info("a related image is not pinned to a digest reference of the same image, and this is required.")
		return false, nil
	}

	// Some environment variables should be passed into the environment using the RELATED_IMAGE_ prefix.
	// This isn't the only way to pass this kind of information to the controller, but it is the suggested way in our
	// documentation.
	deploymentSpecs := make([]appsv1.DeploymentSpec, 0, len(csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs))
	for _, ds := range csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
		deploymentSpecs = append(deploymentSpecs, ds.Spec)
	}
	relatedImagesInContainerEnvironment := libcsv.RelatedImageReferencesInEnvironment(deploymentSpecs...)
	if len(relatedImagesInContainerEnvironment) == 0 {
		logger.Info("no environment variables prefixed with \"RELATED_IMAGE_\" were found in your operator's container definitions. These are expected to pass through values into your controller's runtime environment.")
		return false, nil
	}

	return true, nil
}

func (p FollowsRestrictedNetworkEnablementGuidelines) Name() string {
	return "FollowsRestrictedNetworkEnablementGuidelines"
}

func (p FollowsRestrictedNetworkEnablementGuidelines) Metadata() check.Metadata {
	return check.Metadata{
		Description: "Checks for indicators that this bundle has implemented guidelines to indicate readiness for running in a disconnected cluster, or a cluster with a restricted network.",
		// TODO: If this check is enforced and no longer optional, we need to identify ways to reduce false failures that may be caused by
		// developers injecting related images in other ways.
		Level:            "optional",
		KnowledgeBaseURL: "https://access.redhat.com/documentation/en-us/red_hat_software_certification/2024/html-single/red_hat_openshift_software_certification_policy_guide/index#con-operator-requirements_openshift-sw-cert-policy-products-managed",
		CheckURL:         "https://access.redhat.com/documentation/en-us/red_hat_software_certification/2024/html-single/red_hat_openshift_software_certification_policy_guide/index#con-operator-requirements_openshift-sw-cert-policy-products-managed",
	}
}

func (p FollowsRestrictedNetworkEnablementGuidelines) Help() check.HelpText {
	return check.HelpText{
		Message:    "Check for the implementation of guidelines indicating operator readiness for environments with restricted networking.",
		Suggestion: "If consumers of your operator may need to do so on a restricted network, implement the guidelines outlines in OCP documentation for your cluster version, such as https://docs.openshift.com/container-platform/4.11/operators/operator_sdk/osdk-generating-csvs.html#olm-enabling-operator-for-restricted-network_osdk-generating-csvs for OCP 4.11",
	}
}

func (p FollowsRestrictedNetworkEnablementGuidelines) RequiredFilePatterns() []string {
	return bundle.BundleFiles
}
