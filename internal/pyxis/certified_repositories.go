package pyxis

import (
	"context"
	"fmt"
	"net/http"
	"time"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/shurcooL/graphql"
)

// RepositoryLifecycleForLayers takes uncompressedLayerHashes and queries to a Red Hat Pyxis,
// returning existing cert repositories from registry.access.redhat.com that contain any of the
// IDs as its uncompressed top layer id.
func (p *pyxisClient) CertifiedRepositoriesForLayers(ctx context.Context, uncompressedLayerHashes []cranev1.Hash) ([]CertRepository, error) {
	filteredHashes := filterExcludedLayers(uncompressedLayerHashes)

	layerIds := make([]graphql.String, 0, len(filteredHashes))
	for _, layer := range filteredHashes {
		layerIds = append(layerIds, graphql.String(layer.String()))
	}

	// find images graphql query, including container repositories as edges, to reduce the number of calls to pyxis
	var query struct {
		FindImages struct {
			ContainerImage []struct {
				ID                     graphql.String `graphql:"_id"`
				UncompressedTopLayerID graphql.String `graphql:"uncompressed_top_layer_id"`
				Repositories           []struct {
					Registry   graphql.String
					Repository graphql.String
					Edges      *struct {
						Repository *struct {
							ContainerRepository *struct {
								ID                graphql.String   `graphql:"_id"`
								Registry          graphql.String   `graphql:"registry"`
								Repository        graphql.String   `graphql:"repository"`
								EOLDate           *graphql.String  `graphql:"eol_date"`
								ReleaseCategories []graphql.String `graphql:"release_categories"`
							} `graphql:"data"`
							Error *struct {
								Status graphql.Int    `graphql:"status"`
								Detail graphql.String `graphql:"detail"`
							} `graphql:"error"`
						} `graphql:"repository"`
					} `graphql:"edges"`
				} `graphql:"repositories"`
			} `graphql:"data"`
			Error *struct {
				Status graphql.Int    `graphql:"status"`
				Detail graphql.String `graphql:"detail"`
			} `graphql:"error"`
			Total graphql.Int
			Page  graphql.Int
			// filter to make sure we get exact results
		} `graphql:"find_images(filter:{and:[{repositories:{registry:{in:$registries}}}{uncompressed_top_layer_id:{in:$contImageLayers}}]})"`
	}

	variables := map[string]any{
		"contImageLayers": layerIds,
		"registries":      []graphql.String{accessRegistry},
	}

	httpClient, ok := p.Client.(*http.Client)
	if !ok {
		//coverage:ignore
		return nil, fmt.Errorf("client could not be used as http.Client")
	}
	client := graphql.NewClient(p.getPyxisGraphqlURL(), httpClient)

	err := client.Query(ctx, &query, variables)
	if err != nil {
		//coverage:ignore
		return nil, fmt.Errorf("error while executing cert image query: %w", err)
	}

	if query.FindImages.Error != nil {
		return nil, fmt.Errorf(
			"cert image query returned an error: status %d: %s",
			query.FindImages.Error.Status,
			query.FindImages.Error.Detail,
		)
	}

	certRepositories := make([]CertRepository, 0)
	var edgeErr error
	// keep one edge error while allowing other repository edges to be processed.
	recordEdgeError := func(err error) {
		if edgeErr == nil {
			edgeErr = err
		}
	}

	for _, image := range query.FindImages.ContainerImage {
		for _, imageRepository := range image.Repositories {
			// ensure registry edge data only comes from registry.access.redhat.com
			if imageRepository.Registry != accessRegistry {
				continue
			}

			if imageRepository.Edges == nil || imageRepository.Edges.Repository == nil {
				recordEdgeError(fmt.Errorf("repository data for image %s and repository %s were null", image.ID, imageRepository.Repository))
				continue
			}

			repository := imageRepository.Edges.Repository
			if repository.Error != nil {
				recordEdgeError(fmt.Errorf(
					"repository query returned an error for image %s and repository %s: status %d: %s",
					image.ID,
					imageRepository.Repository,
					repository.Error.Status,
					repository.Error.Detail,
				))
				continue
			}

			if repository.ContainerRepository == nil {
				recordEdgeError(fmt.Errorf("container repository for image %s and repository %s were null", image.ID, imageRepository.Repository))
				continue
			}

			containerRepository := repository.ContainerRepository

			var eolDate *time.Time
			// A missing EOL date means the repository has no scheduled EOL.
			if value := containerRepository.EOLDate; value != nil && *value != "" {
				parsed, err := time.Parse(time.RFC3339, string(*value))
				if err != nil {
					return nil, fmt.Errorf("invalid RFC3339 eol_date %q for image %s and repository %s: %w", *value, image.ID, imageRepository.Repository, err)
				}

				eolDate = &parsed
			}

			// converting from graphql string array to string array
			releaseCategories := make([]string, len(containerRepository.ReleaseCategories))
			for i := range containerRepository.ReleaseCategories {
				releaseCategories[i] = string(containerRepository.ReleaseCategories[i])
			}

			certRepositories = append(certRepositories, CertRepository{
				ID:                string(containerRepository.ID),
				Registry:          string(containerRepository.Registry),
				Repository:        string(containerRepository.Repository),
				EOLDate:           eolDate,
				ReleaseCategories: releaseCategories,
			})
		}
	}

	// return an edge error only when no repository data was usable.
	if len(certRepositories) == 0 && edgeErr != nil {
		return nil, edgeErr
	}

	return certRepositories, nil
}
