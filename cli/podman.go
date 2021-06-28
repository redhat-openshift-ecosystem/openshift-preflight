package cli

type ImagePullOptions struct {
	LogLevel string
}

type ImagePullReport struct {
	StdoutErr string
}

type ImageRunOptions struct {
	EntryPoint     string
	EntryPointArgs []string
	Image          string
	LogLevel       string
}

type ImageRunReport struct {
	Stdout string
	Stderr string
}

type ImageSaveOptions struct {
	LogLevel string
}

type PodmanEngine interface {
	Pull(rawImage string, opts ImagePullOptions) (*ImagePullReport, error)
	Run(opts ImageRunOptions) (*ImageRunReport, error)
	Save(nameOrID string, tags []string, opts ImageSaveOptions) error
}
