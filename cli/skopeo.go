package cli

type SkopeoListTagsReport struct {
	Stdout string
	Stderr string
	Tags   []string
}

// SkopeoInspectReport contains information about the execution of a
// skopeo inspect.
type SkopeoInspectReport struct {
	Stdout string
	Stderr string
	// Blob represents the JSON response from Skopeo.
	// This response could vary in format so it is up to the caller
	// to serialize this as they expect based on SkopeoInspectOptions
	Blob []byte
}

// SkopeoInspectOptions represent options to pass to `skopeo inspect` command
// invocations
type SkopeoInspectOptions struct {
	Raw bool
}

type SkopeoEngine interface {
	ListTags(rawImage string) (*SkopeoListTagsReport, error)
	InspectImage(rawImage string, inspectOptions SkopeoInspectOptions) (*SkopeoInspectReport, error)
}
