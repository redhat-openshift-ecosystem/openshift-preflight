package certification

import (
	"time"
)

type Result struct {
	Check
	ElapsedTime time.Duration
}

type Results struct {
	TestedImage       string
	PassedOverall     bool
	TestedOn          string
	CertificationHash string
	Passed            []Result
	Failed            []Result
	Errors            []Result
}
