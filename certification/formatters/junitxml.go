package formatters

import (
	"encoding/xml"
	"fmt"

	"github.com/komish/preflight/certification/errors"
	"github.com/komish/preflight/certification/runtime"
)

type JUnitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []JUnitTestSuite `xml:"testsuite"`
}

type JUnitTestSuite struct {
	XMLName    xml.Name        `xml:"testsuite"`
	Tests      int             `xml:"tests,attr"`
	Failures   int             `xml:"failures,attr"`
	Time       string          `xml:"time,attr"`
	Name       string          `xml:"name,attr"`
	Properties []JUnitProperty `xml:"properties>property,omitempty"`
	TestCases  []JUnitTestCase `xml:"testcase"`
}

type JUnitTestCase struct {
	XMLName     xml.Name          `xml:"testcase"`
	Classname   string            `xml:"classname,attr"`
	Name        string            `xml:"name,attr"`
	Time        string            `xml:"time,attr"`
	SkipMessage *JUnitSkipMessage `xml:"skipped,omitempty"`
	Failure     *JUnitFailure     `xml:"failure,omitempty"`
	SystemOut   string            `xml:"system-out,omitempty"`
}

type JUnitSkipMessage struct {
	Message string `xml:"message,attr"`
}

type JUnitProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type JUnitFailure struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"`
}

func junitXMLFormatter(r runtime.Results) ([]byte, error) {
	response := getResponse(r)
	suites := JUnitTestSuites{}
	testsuite := JUnitTestSuite{
		Tests:      len(r.Errors) + len(r.Failed) + len(r.Passed),
		Failures:   len(r.Errors) + len(r.Failed),
		Time:       "0s",
		Name:       "Red Hat Certification",
		Properties: []JUnitProperty{},
		TestCases:  []JUnitTestCase{},
	}

	for _, result := range r.Passed {
		testCase := JUnitTestCase{
			Classname: response.Image,
			Name:      result.Name(),
			Time:      "0s",
			Failure:   nil,
		}
		testsuite.TestCases = append(testsuite.TestCases, testCase)
	}

	for _, result := range append(r.Errors, r.Failed...) {
		testCase := JUnitTestCase{
			Classname: response.Image,
			Name:      result.Name(),
			Time:      "0s",
			Failure: &JUnitFailure{
				Message:  "Failed",
				Type:     "",
				Contents: fmt.Sprintf("%s: Suggested Fix: %s", result.Help().Message, result.Help().Suggestion),
			},
		}
		testsuite.TestCases = append(testsuite.TestCases, testCase)
	}

	suites.Suites = append(suites.Suites, testsuite)

	bytes, err := xml.MarshalIndent(suites, "", "\t")
	if err != nil {
		o := fmt.Errorf("%w with formatter %s: %s",
			errors.ErrFormattingResults,
			"junitxml",
			err,
		)

		return nil, o
	}

	return bytes, nil
}
