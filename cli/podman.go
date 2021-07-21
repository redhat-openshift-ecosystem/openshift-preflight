package cli

type ImageInspectOptions struct {
	LogLevel string
}

type ImageInspectReport struct {
	Images []PodmanImage
}

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
	LogLevel    string
	Image       string
	Destination string
}

type PodmanImage struct {
	Id     string
	Config PodmanImageConfig
	RootFS PodmanRootFS
}

type PodmanImageConfig struct {
	Labels map[string]string
}

type PodmanRootFS struct {
	Type   string
	Layers []string
}

type ImageScanReport struct {
	Stdout string
	Stderr string
}

type PodmanCopyReport struct {
	Stdout string
	Stderr string
}

type PodmanCreateReport struct {
	Stdout      string
	Stderr      string
	ContainerID string
}

type PodmanCreateOptions struct {
	// Entrypoint is an explicit entrypoint to pass to the Create command. Leaving this
	// empty will not add the --entrypoint flag to the command invocation.
	Entrypoint string
}

type PodmanRemoveReport struct {
	Stdout string
	Stderr string
}

type PodmanEngine interface {
	InspectImage(rawImage string, opts ImageInspectOptions) (*ImageInspectReport, error)
	Pull(rawImage string, opts ImagePullOptions) (*ImagePullReport, error)
	Run(opts ImageRunOptions) (*ImageRunReport, error)
	Save(nameOrID string, tags []string, opts ImageSaveOptions) error
	ScanImage(image string) (*ImageScanReport, error)
	Create(rawImage string, opts *PodmanCreateOptions) (*PodmanCreateReport, error)
	CopyFrom(containerID, sourcePath, destinationPath string) (*PodmanCopyReport, error)
	Remove(containerID string) (*PodmanRemoveReport, error)
}
