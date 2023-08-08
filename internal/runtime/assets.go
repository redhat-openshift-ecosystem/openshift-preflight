package runtime

import (
	"context"
	"fmt"
	goruntime "runtime"
	"strings"

	"github.com/go-logr/logr"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/authn"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"

	"github.com/google/go-containerregistry/pkg/crane"
	cranev1 "github.com/google/go-containerregistry/pkg/v1"
)

// images maps the images use by preflight with their purpose.
//
// these should have accessor functions made available if they are
// to be used outside of this package.
var images = map[string]string{
	// operator policy, operator-sdk scorecard
	"scorecard": "quay.io/operator-framework/scorecard-test:v1.31.0",
}

// imageList takes the images mapping and represents them using just
// the image URIs.
func imageList(ctx context.Context) []string {
	logger := logr.FromContextOrDiscard(ctx)
	options := []crane.Option{
		crane.WithContext(ctx),
		crane.WithAuthFromKeychain(authn.PreflightKeychain(ctx)),
		crane.WithPlatform(&cranev1.Platform{
			OS: "linux",
			// This remains the runtime arch, as we don't specify the arch for operators
			Architecture: goruntime.GOARCH,
		}),
	}

	imageList := make([]string, 0, len(images))

	for _, image := range images {
		base := strings.Split(image, ":")[0]
		digest, err := crane.Digest(image, options...)
		if err != nil {
			logger.Error(fmt.Errorf("could not retrieve image digest: %w", err), "crane error")
			// Skip this entry
			continue
		}
		imageList = append(imageList, fmt.Sprintf("%s@%s", base, digest))
	}

	return imageList
}

// Assets returns a full collection of assets used in Preflight.
func Assets(ctx context.Context) AssetData {
	return AssetData{
		Images: imageList(ctx),
	}
}

// ScorecardImage returns the container image used for OperatorSDK
// Scorecard based checks. If userProvidedScorecardImage is set, it is
// returned, otherwise, the default is returned.
func ScorecardImage(ctx context.Context, userProvidedScorecardImage string) string {
	logger := logr.FromContextOrDiscard(ctx)
	if userProvidedScorecardImage != "" {
		logger.V(log.DBG).Info("user provided scorecard test image", "image", userProvidedScorecardImage)
		return userProvidedScorecardImage
	}
	return images["scorecard"]
}

// Assets is the publicly accessible representation of Preflight's
// used assets. This struct will be serialized to JSON and presented
// to the end-user when requested.
type AssetData struct {
	Images []string `json:"images"`
}
