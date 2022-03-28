package pyxis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	log "github.com/sirupsen/logrus"
)

func (p *pyxisClient) CheckRedHatLayers(ctx context.Context, layerHashes []cranev1.Hash) ([]CertImage, error) {
	layerIds := make([]string, 0, len(layerHashes))
	for _, layer := range layerHashes {
		layerIds = append(layerIds, layer.String())
	}
	log.Tracef("the layerIds passed to pyxis are %s", layerIds)

	pyxisQuery := url.QueryEscape(fmt.Sprintf("repositories.registry==registry.access.redhat.com and uncompressed_top_layer_id=in=(%s)", strings.Join(layerIds, ",")))

	req, err := p.newRequest(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s?filter=%s", p.getPyxisUrl("images"), pyxisQuery),
		nil,
	)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	log.Tracef("URL is %s", req.URL)
	resp, err := p.Client.Do(req)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if !checkStatus(resp.StatusCode) {
		log.Errorf("%s: %s", "received non 200 status code in CheckRedHatLayers", string(body))
		return nil, errors.ErrNon200StatusCode
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
