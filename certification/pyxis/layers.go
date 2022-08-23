package pyxis

import (
	"context"
	"fmt"
	"net/http"
	"time"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/shurcooL/graphql"
)

// CertifiedImagesContainingLayers takes uncompressedLayerHashes and queries to a Red Hat Pyxis,
// returning existing certified images from registry.access.redhat.com that contain any of the
// IDs as its uncompressed top layer id.
func (p *pyxisClient) CertifiedImagesContainingLayers(ctx context.Context, uncompressedLayerHashes []cranev1.Hash) ([]CertImage, error) {
	layerIds := make([]graphql.String, 0, len(uncompressedLayerHashes))
	for _, layer := range uncompressedLayerHashes {
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
	variables := map[string]interface{}{
		"contImageLayers": layerIds,
		"registries":      []graphql.String{"registry.access.redhat.com"},
	}

	// make our query
	httpClient, ok := p.Client.(*http.Client)
	if !ok {
		return nil, fmt.Errorf("client could not be used as http.Client")
	}
	client := graphql.NewClient(p.getPyxisGraphqlURL(), httpClient)

	err := client.Query(ctx, &query, variables)
	if err != nil {
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
