package formatters

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"

	"gotest.tools/v3/assert"
)

func TestGenericJSONFormatter(t *testing.T) {
	generateTestResults := func(image string, passed bool) certification.Results {
		return certification.Results{
			TestedImage:   image,
			PassedOverall: passed,
			Passed: []certification.Result{
				{
					Check:       check.NewGenericCheck("passed1", nil, check.Metadata{}, check.HelpText{}, nil),
					ElapsedTime: 1000 * time.Millisecond,
				},
			},
			Failed: []certification.Result{
				{
					Check:       check.NewGenericCheck("failed1", nil, check.Metadata{}, check.HelpText{}, nil),
					ElapsedTime: 1001 * time.Millisecond,
				},
			},
		}
	}

	testCases := []struct {
		results              certification.Results
		marshalIndentFailure bool
		expectedErrString    string
	}{
		{
			results:              generateTestResults("image1", true),
			marshalIndentFailure: false,
		},
		{
			results:              generateTestResults("image2", false),
			marshalIndentFailure: false,
		},
		{
			results:              generateTestResults("image3", true),
			marshalIndentFailure: true, // failure to json.MarshalIndent
			expectedErrString:    "this is an error",
		},
	}

	for _, tc := range testCases {
		// Patch the function if we expect an error
		if tc.marshalIndentFailure {
			jsonMarshalIndent = func(v any, prefix, indent string) ([]byte, error) {
				return nil, errors.New("this is an error")
			}
		} else {
			jsonMarshalIndent = json.MarshalIndent
		}

		// Run the function
		funcOutput, err := genericJSONFormatter(context.TODO(), tc.results)

		if err == nil {
			// Marshal the response JSON back into an object
			var testResponseObj UserResponse
			assert.NilError(t, json.Unmarshal(funcOutput, &testResponseObj))

			// Assertions
			assert.Equal(t, tc.results.TestedImage, testResponseObj.Image)
			assert.Equal(t, tc.results.PassedOverall, testResponseObj.Passed)

			for index, i := range tc.results.Passed {
				assert.Equal(t, i.Name(), testResponseObj.Results.Passed[index].Name)
				assert.Equal(t, float64(i.ElapsedTime/time.Millisecond), testResponseObj.Results.Passed[index].ElapsedTime)
			}
			for index, i := range tc.results.Failed {
				assert.Equal(t, i.Name(), testResponseObj.Results.Failed[index].Name)
				assert.Equal(t, float64(i.ElapsedTime/time.Millisecond), testResponseObj.Results.Failed[index].ElapsedTime)
			}
		} else {
			assert.Equal(t, true, strings.Contains(err.Error(), tc.expectedErrString))
		}
	}
}

func TestGenericXMLFormatter(t *testing.T) {
	generateTestResults := func(image string, passed bool) certification.Results {
		return certification.Results{
			TestedImage:   image,
			PassedOverall: passed,
			Passed: []certification.Result{
				{
					Check:       check.NewGenericCheck("passed1", nil, check.Metadata{}, check.HelpText{}, nil),
					ElapsedTime: 1000 * time.Millisecond,
				},
			},
			Failed: []certification.Result{
				{
					Check:       check.NewGenericCheck("failed1", nil, check.Metadata{}, check.HelpText{}, nil),
					ElapsedTime: 1001 * time.Millisecond,
				},
			},
		}
	}

	testCases := []struct {
		results              certification.Results
		marshalIndentFailure bool
		expectedErrString    string
	}{
		{
			results:              generateTestResults("image1", true),
			marshalIndentFailure: false,
		},
		{
			results:              generateTestResults("image2", false),
			marshalIndentFailure: false,
		},
		{
			results:              generateTestResults("image3", true),
			marshalIndentFailure: true, // failure to xml.MarshalIndent
			expectedErrString:    "this is an error",
		},
	}

	for _, tc := range testCases {
		// Patch the function if we expect an error
		if tc.marshalIndentFailure {
			xmlMarshalIndent = func(v any, prefix, indent string) ([]byte, error) {
				return nil, errors.New("this is an error")
			}
		} else {
			xmlMarshalIndent = xml.MarshalIndent
		}

		// Run the function
		funcOutput, err := genericXMLFormatter(context.TODO(), tc.results)

		if err == nil {
			// Marshal the response XML back into an object
			var testResponseObj UserResponse
			assert.NilError(t, xml.Unmarshal(funcOutput, &testResponseObj))

			// Assertions
			assert.Equal(t, tc.results.TestedImage, testResponseObj.Image)
			assert.Equal(t, tc.results.PassedOverall, testResponseObj.Passed)

			for index, i := range tc.results.Passed {
				assert.Equal(t, i.Name(), testResponseObj.Results.Passed[index].Name)
				assert.Equal(t, float64(i.ElapsedTime/time.Millisecond), testResponseObj.Results.Passed[index].ElapsedTime)
			}
			for index, i := range tc.results.Failed {
				assert.Equal(t, i.Name(), testResponseObj.Results.Failed[index].Name)
				assert.Equal(t, float64(i.ElapsedTime/time.Millisecond), testResponseObj.Results.Failed[index].ElapsedTime)
			}
		} else {
			assert.Equal(t, true, strings.Contains(err.Error(), tc.expectedErrString))
		}
	}
}
