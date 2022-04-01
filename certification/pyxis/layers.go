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

func (p *pyxisClient) CheckRedHatLayers(ctx context.Context, layerHashes []cranev1.Hash) ([]CertImage, error) {
	layerIds := make([]graphql.String, 0, len(layerHashes))
	for _, layer := range layerHashes {
		layerIds = append(layerIds, graphql.String(layer.String()))
	}

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
		} `graphql:"find_images(filter: {and:[{repositories:{registry:{in:$registries}}}{uncompressed_top_layer_id:{in:$contImageLayers}}]})"`
	}

	variables := map[string]interface{}{
		"contImageLayers": layerIds,
		"registries":      []graphql.String{"registry.access.redhat.com"},
	}

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
