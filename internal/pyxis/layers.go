package pyxis

import (
	"context"
	"fmt"
	"net/http"
	"time"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/shurcooL/graphql"
)

// filterExcludedLayers removes layer hashes that are known to be common or
// problematic when matching up against a base image.
func filterExcludedLayers(uncompressedLayerHashes []cranev1.Hash) []cranev1.Hash {
	excludedHashes := map[string]struct{}{
		// This hash represents an empty layer, where the contents is only the
		// tar end-of-stream marker 1kb of zeros (dd if=/dev/zero bs=1024
		// count=1 2>/dev/null | sha256sum).
		//
		// Legacy build tools would add this before "empty_layer" became an
		// option for operations like LABEL, ENV, etc. that did not modify the
		// filesystem. There are also more modern cases where we see this behavior
		// e.g. https://github.com/containers/buildah/issues/6860
		//
		// Because it is effectively an empty layer, we do not want to match
		// against base images that also have an empty base layer as their final
		// layer diff (any image can have an empty layer).
		//
		// For this reason, we will not send this to the backend to find an
		// image that contains this layer
		"sha256:5f70bf18a086007016e948b04aed3b82103a36bea41755b6cddfaf10ace3c6ef": {},
	}

	filtered := make([]cranev1.Hash, 0, len(uncompressedLayerHashes))
	for _, layer := range uncompressedLayerHashes {
		if _, isExcluded := excludedHashes[layer.String()]; !isExcluded {
			filtered = append(filtered, layer)
		}
	}

	return filtered
}

// CertifiedImagesContainingLayers takes uncompressedLayerHashes and queries to a Red Hat Pyxis,
// returning existing certified images from registry.access.redhat.com that contain any of the
// IDs as its uncompressed top layer id.
func (p *pyxisClient) CertifiedImagesContainingLayers(ctx context.Context, uncompressedLayerHashes []cranev1.Hash) ([]CertImage, error) {
	filteredHashes := filterExcludedLayers(uncompressedLayerHashes)

	layerIds := make([]graphql.String, 0, len(filteredHashes))
	for _, layer := range filteredHashes {
		layerIds = append(layerIds, graphql.String(layer.String()))
	}

	// our graphQL query
	var query struct {
		FindImages struct {
			ContainerImage []struct {
				UncompressedTopLayerID graphql.String `graphql:"uncompressed_top_layer_id"`
				ID                     graphql.String `graphql:"_id"`
				FreshnessGrades        []struct {
					Grade     graphql.String `graphql:"grade"`
					StartDate graphql.String `graphql:"start_date"`
					EndDate   graphql.String `graphql:"end_date"`
				} `graphql:"freshness_grades"`
			} `graphql:"data"`
			Error struct {
				Status graphql.Int    `graphql:"status"`
				Detail graphql.String `graphql:"detail"`
			} `graphql:"error"`
			Total graphql.Int
			Page  graphql.Int
			// filter to make sure we get exact results
		} `graphql:"find_images(filter: {and:[{repositories:{registry:{in:$registries}}}{uncompressed_top_layer_id:{in:$contImageLayers}}]})"`
	}

	// variables to feed to our graphql filter
	variables := map[string]any{
		"contImageLayers": layerIds,
		"registries":      []graphql.String{"registry.access.redhat.com"},
	}

	// make our query
	httpClient, ok := p.Client.(*http.Client)
	if !ok {
		//coverage:ignore
		return nil, fmt.Errorf("client could not be used as http.Client")
	}
	client := graphql.NewClient(p.getPyxisGraphqlURL(), httpClient)

	err := client.Query(ctx, &query, variables)
	if err != nil {
		//coverage:ignore
		return nil, fmt.Errorf("error while executing layers query: %v", err)
	}

	images := make([]CertImage, 0, len(query.FindImages.ContainerImage))
	for _, image := range query.FindImages.ContainerImage {
		freshnessGrades := make([]FreshnessGrade, 0, len(image.FreshnessGrades))
		for _, grade := range image.FreshnessGrades {
			startDate, _ := time.Parse(time.RFC3339, string(grade.StartDate))
			endDate, _ := time.Parse(time.RFC3339, string(grade.EndDate))
			freshnessGrades = append(freshnessGrades, FreshnessGrade{
				Grade:     string(grade.Grade),
				StartDate: startDate,
				EndDate:   endDate,
			})
		}
		images = append(images, CertImage{
			ID:                     string(image.ID),
			UncompressedTopLayerID: string(image.UncompressedTopLayerID),
			FreshnessGrades:        freshnessGrades,
		})
	}

	return images, nil
}
