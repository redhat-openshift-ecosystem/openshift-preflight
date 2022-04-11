package pyxis

import (
	"context"
	"fmt"
	"net/http"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/hasura/go-graphql-client"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	log "github.com/sirupsen/logrus"
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
				UncompressedTopLayerId graphql.String `graphql:"uncompressed_top_layer_id"`
				ID                     graphql.String `graphql:"_id"`
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
		return nil, errors.ErrInvalidHttpClient
	}
	client := graphql.NewClient(p.getPyxisGraphqlUrl(), httpClient).WithDebug(true)

	err := client.Query(ctx, &query, variables)
	if err != nil {
		log.Error(fmt.Errorf("%w: %s", errors.ErrInvalidGraphqlQuery, err))
		return nil, err
	}

	images := make([]CertImage, 0, len(query.FindImages.ContainerImage))
	for _, image := range query.FindImages.ContainerImage {
		images = append(images, CertImage{
			ID:                     string(image.ID),
			UncompressedTopLayerId: string(image.UncompressedTopLayerId),
		})
	}

	return images, nil
}
