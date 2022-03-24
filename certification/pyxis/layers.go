package pyxis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	log "github.com/sirupsen/logrus"
)

func (p *pyxisEngine) CheckRedHatLayers(ctx context.Context, layerHashes []cranev1.Hash) ([]CertImage, error) {
	layerIds := make([]string, 0, len(layerHashes))
	for _, layer := range layerHashes {
		layerIds = append(layerIds, layer.String())
	}
	log.Tracef("the layerIds passed to pyxis are %s", layerIds)

	pyxisQuery := url.QueryEscape(fmt.Sprintf("repositories.registry=in=(registry.access.redhat.com) and uncompressed_top_layer_id=in=(%s)", strings.Join(layerIds, ",")))

	req, err := p.newRequestWithApiToken(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s?filter=%s", getPyxisUrl("images"), pyxisQuery),
		nil,
	)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if resp.StatusCode != 200 {
		err := fmt.Sprintf("Recieved http status %s instead of 200 OK", resp.Status)
		log.Error("Unexpected Status Code", err)
		return nil, errors.New(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	log.Tracef("query response from pyxis %s", string(body))

	type imageList struct {
		Images []CertImage `json:"data"`
	}

	var images imageList
	if err := json.Unmarshal(body, &images); err != nil {
		log.Error(err)
		return nil, err
	}

	return images.Images, nil
}
