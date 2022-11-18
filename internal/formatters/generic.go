package formatters

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
)

var (
	jsonMarshalIndent = json.MarshalIndent
	xmlMarshalIndent  = xml.MarshalIndent
)

// genericJSONFormatter is a FormatterFunc that formats results as JSON
func genericJSONFormatter(ctx context.Context, r certification.Results) ([]byte, error) {
	response := getResponse(r)

	responseJSON, err := jsonMarshalIndent(response, "", "    ")
	if err != nil {
		e := fmt.Errorf("error formatting results with formatter %s: %w",
			"json",
			err,
		)

		return nil, e
	}

	return responseJSON, nil
}

// genericXMLFormatter is a FormatterFunc that formats results as XML
func genericXMLFormatter(ctx context.Context, r certification.Results) ([]byte, error) {
	response := getResponse(r)

	responseXML, err := xmlMarshalIndent(response, "", "    ")
	if err != nil {
		e := fmt.Errorf("error formatting results with formatter %s: %w",
			"xml",
			err,
		)

		return nil, e
	}

	return responseXML, nil
}
