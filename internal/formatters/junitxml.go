package formatters

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
)

type JUnitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []JUnitTestSuite `xml:"testsuite"`
}

type JUnitTestSuite struct {
	XMLName    xml.Name        `xml:"testsuite"`
	Tests      int             `xml:"tests,attr"`
	Failures   int             `xml:"failures,attr"`
	Warnings   int             `xml:"warnings,attr"`
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
	Failure     *JUnitMessage     `xml:"failure,omitempty"`
	Warning     *JUnitMessage     `xml:"warning,omitempty"`
	SystemOut   string            `xml:"system-out,omitempty"`
	Message     string            `xml:",chardata"`
}

type JUnitSkipMessage struct {
	Message string `xml:"message,attr"`
}

type JUnitProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type JUnitMessage struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"`
}

func junitXMLFormatter(_ context.Context, r certification.Results) ([]byte, error) {
	response := getResponse(r)
	suites := JUnitTestSuites{}
	testsuite := JUnitTestSuite{
		Tests:      len(r.Errors) + len(r.Failed) + len(r.Passed) + len(r.Warned),
		Failures:   len(r.Errors) + len(r.Failed),
		Warnings:   len(r.Warned),
		Time:       "0s",
		Name:       "Red Hat Certification",
		Properties: []JUnitProperty{},
		TestCases:  []JUnitTestCase{},
	}

	totalDuration := time.Duration(0)
	for _, result := range r.Passed {
		testCase := JUnitTestCase{
			Classname: response.Image,
			Name:      result.Name(),
			Time:      fmt.Sprintf("%f", result.ElapsedTime.Seconds()),
			Failure:   nil,
			Message:   result.Metadata().Description,
		}
		testsuite.TestCases = append(testsuite.TestCases, testCase)
		totalDuration += result.ElapsedTime
	}

	for _, result := range append(r.Errors, r.Failed...) {
		testCase := JUnitTestCase{
			Classname: response.Image,
			Name:      result.Name(),
			Time:      result.ElapsedTime.String(),
			Failure: &JUnitMessage{
				Message:  "Failed",
				Type:     "",
				Contents: fmt.Sprintf("%s: Suggested Fix: %s", result.Help().Message, result.Help().Suggestion),
			},
		}
		testsuite.TestCases = append(testsuite.TestCases, testCase)
		totalDuration += result.ElapsedTime
	}

	for _, result := range r.Warned {
		testCase := JUnitTestCase{
			Classname: response.Image,
			Name:      result.Name(),
			Time:      result.ElapsedTime.String(),
			Warning: &JUnitMessage{
				Message:  "Warn",
				Type:     "",
				Contents: fmt.Sprintf("%s: Suggested Fix: %s", result.Help().Message, result.Help().Suggestion),
			},
		}
		testsuite.TestCases = append(testsuite.TestCases, testCase)
		totalDuration += result.ElapsedTime
	}

	testsuite.Time = fmt.Sprintf("%f", totalDuration.Seconds())
	suites.Suites = append(suites.Suites, testsuite)

	bytes, err := xml.MarshalIndent(suites, "", "\t")
	if err != nil {
		o := fmt.Errorf("error formatting results with formatter %s: %v",
			"junitxml",
			err,
		)

		return nil, o
	}

	return bytes, nil
}
