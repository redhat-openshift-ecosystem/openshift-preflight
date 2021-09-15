package formatters

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"gotest.tools/v3/assert"
)

func TestGenericJSONFormatter(t *testing.T) {
	generateTestResults := func(image string, passed bool) runtime.Results {
		return runtime.Results{
			TestedImage:   image,
			PassedOverall: passed,
			Passed: []runtime.Result{
				{
					Check:       certification.NewGenericCheck("passed1", nil, certification.Metadata{}, certification.HelpText{}),
					ElapsedTime: time.Duration(1000 * time.Millisecond),
				},
			},
			Failed: []runtime.Result{
				{
					Check:       certification.NewGenericCheck("failed1", nil, certification.Metadata{}, certification.HelpText{}),
					ElapsedTime: time.Duration(1001 * time.Millisecond),
				},
			},
		}
	}

	testCases := []struct {
		results              runtime.Results
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
			jsonMarshalIndent = func(v interface{}, prefix, indent string) ([]byte, error) {
				return nil, errors.New("this is an error")
			}
		} else {
			jsonMarshalIndent = json.MarshalIndent
		}

		// Run the function
		funcOutput, err := genericJSONFormatter(tc.results)

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
	generateTestResults := func(image string, passed bool) runtime.Results {
		return runtime.Results{
			TestedImage:   image,
			PassedOverall: passed,
			Passed: []runtime.Result{
				{
					Check:       certification.NewGenericCheck("passed1", nil, certification.Metadata{}, certification.HelpText{}),
					ElapsedTime: time.Duration(1000 * time.Millisecond),
				},
			},
			Failed: []runtime.Result{
				{
					Check:       certification.NewGenericCheck("failed1", nil, certification.Metadata{}, certification.HelpText{}),
					ElapsedTime: time.Duration(1001 * time.Millisecond),
				},
			},
		}
	}

	testCases := []struct {
		results              runtime.Results
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
			xmlMarshalIndent = func(v interface{}, prefix, indent string) ([]byte, error) {
				return nil, errors.New("this is an error")
			}
		} else {
			xmlMarshalIndent = xml.MarshalIndent
		}

		// Run the function
		funcOutput, err := genericXMLFormatter(tc.results)

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
