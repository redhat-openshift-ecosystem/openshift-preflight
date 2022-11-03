package runtime

import (
	"context"
	"fmt"
	goruntime "runtime"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/authn"

	"github.com/google/go-containerregistry/pkg/crane"
	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	log "github.com/sirupsen/logrus"
)

// images maps the images use by preflight with their purpose.
//
// these should have accessor functions made available if they are
// to be used outside of this package.
var images = map[string]string{
	// operator policy, operator-sdk scorecard
	"scorecard": "quay.io/operator-framework/scorecard-test:v1.22.2",
}

// imageList takes the images mapping and represents them using just
// the image URIs.
func imageList(ctx context.Context) []string {
	options := []crane.Option{
		crane.WithContext(ctx),
		crane.WithAuthFromKeychain(authn.PreflightKeychain()),
		crane.WithPlatform(&cranev1.Platform{
			OS:           "linux",
			Architecture: goruntime.GOARCH,
		}),
	}

	imageList := make([]string, 0, len(images))

	for _, image := range images {
		base := strings.Split(image, ":")[0]
		digest, err := crane.Digest(image, options...)
		if err != nil {
			log.Error(fmt.Errorf("could not retrieve image digest: %w", err))
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
func ScorecardImage(userProvidedScorecardImage string) string {
	if userProvidedScorecardImage != "" {
		log.Debugf("Using %s as the scorecard test image", userProvidedScorecardImage)
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
