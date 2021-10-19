package runtime

var (

	// images maps the images use by preflight with their purpose.
	//
	// these should have accessor functions made available if they are
	// to be used outside of this package.
	//
	// These images should also all be referenced using digests over tags
	// to enable disconnected environments.
	images = map[string]string{
		// operator policy, operator-sdk scorecard
		// quay.io/operator-framework/scorecard-test:v1.13.1
		"scorecard": "quay.io/operator-framework/scorecard-test@sha256:417d5157a782f72ba83cc613a0a3997cea088fe025c188121f782e93b38bf04e",
	}
)

// imageList takes the images mapping and represents them using just
// the image URIs.
func imageList() []string {
	var imageList = make([]string, len(images))

	i := 0
	for _, image := range images {
		imageList[i] = image
		i++
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
	return images["scorecard"]
}

// Assets is the publicly accessible representation of Preflight's
// used assets. This struct will be serialized to JSON and presented
// to the end-user when requested.
type AssetData struct {
	Images []string `json:"images"`
}
