package formatters

import (
	"encoding/json"
	"encoding/xml"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
)

// genericJSONFormatter is a FormatterFunc that formats results as JSON
func genericJSONFormatter(r runtime.Results) ([]byte, error) {
	response := getResponse(r)

	responseJSON, err := json.MarshalIndent(response, "", "    ")
	if err != nil {
		e := fmt.Errorf("%w with formatter %s: %s",
			errors.ErrFormattingResults,
			"json",
			err,
		)

		return nil, e
	}

	return responseJSON, nil
}

// genericXMLFormatter is a FormatterFunc that formats results as XML
func genericXMLFormatter(r runtime.Results) ([]byte, error) {
	response := getResponse(r)

	responseJSON, err := xml.MarshalIndent(response, "", "    ")
	if err != nil {
		e := fmt.Errorf("%w with formatter %s: %s",
			errors.ErrFormattingResults,
			"json",
			err,
		)

		return nil, e
	}

	return responseJSON, nil
}
