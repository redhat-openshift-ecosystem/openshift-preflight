package bundle

import "errors"

var (
	ErrEmptyAnnotationFile = errors.New("the annotations file was empty")
	ErrTooManyCSVs         = errors.New("more than one CSV file detected in bundle")
)
