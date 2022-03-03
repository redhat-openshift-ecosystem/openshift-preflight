package runtime

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// images maps the images use by preflight with their purpose.
//
// these should have accessor functions made available if they are
// to be used outside of this package.
var images = map[string]string{
	// operator policy, operator-sdk scorecard
	"scorecard": "quay.io/operator-framework/scorecard-test:v1.14.0",
}

// imageList takes the images mapping and represents them using just
// the image URIs.
func imageList() []string {
	imageList := make([]string, 0, len(images))

	for _, image := range images {
		base := strings.Split(image, ":")[0]
		digest, err := crane.Digest(image)
		if err != nil {
			log.Error(err)
			// Skip this entry
			continue
		}
		imageList = append(imageList, fmt.Sprintf("%s@%s", base, digest))
	}

	return imageList
}

// Assets returns a full collection of assets used in Preflight.
func Assets() AssetData {
	return AssetData{
		Images: imageList(),
	}
}

// ScorecardImage returns the container image used for OperatorSDK
// Scorecard based checks.
func ScorecardImage() string {
	scorecardImage := viper.GetString("scorecard_image")
	if scorecardImage != "" {
		log.Infof("Using %s as the scorecard test image", scorecardImage)
		return scorecardImage
	}
	return images["scorecard"]
}

// Assets is the publicly accessible representation of Preflight's
// used assets. This struct will be serialized to JSON and presented
// to the end-user when requested.
type AssetData struct {
	Images []string `json:"images"`
}
