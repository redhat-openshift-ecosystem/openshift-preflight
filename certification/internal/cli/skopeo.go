package cli

type SkopeoListTagsReport struct {
	Stdout string
	Stderr string
	Tags   []string
}

type SkopeoEngine interface {
	ListTags(rawImage string) (*SkopeoListTagsReport, error)
}
