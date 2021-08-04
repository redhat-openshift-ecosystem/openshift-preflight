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
	Stdout   string
	Stderr   string
	ExitCode int
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

type PodmanMountReport struct {
	Stdout   string
	Stderr   string
	MountDir string
}

type PodmanUnmountReport struct {
	Stdout string
	Stderr string
}

type PodmanUnshareReport struct {
	Stdout string
	Stderr string
}

type PodmanUnshareCheckReport struct {
	PodmanUnshareReport
	PassedOverall bool `json:"passed"`
}

type PodmanEngine interface {
	Create(rawImage string, opts *PodmanCreateOptions) (*PodmanCreateReport, error)
	CopyFrom(containerID, sourcePath, destinationPath string) (*PodmanCopyReport, error)
	InspectImage(rawImage string, opts ImageInspectOptions) (*ImageInspectReport, error)
	Mount(containerId string) (*PodmanMountReport, error)
	MountImage(imageID string) (*PodmanMountReport, error)
	Pull(rawImage string, opts ImagePullOptions) (*ImagePullReport, error)
	Remove(containerID string) (*PodmanRemoveReport, error)
	Run(opts ImageRunOptions) (*ImageRunReport, error)
	Save(nameOrID string, tags []string, opts ImageSaveOptions) error
	ScanImage(image string) (*ImageScanReport, error)
	Unmount(containerId string) (*PodmanUnmountReport, error)
	UnmountImage(imageID string) (*PodmanUnmountReport, error)
	Unshare(env map[string]string, command ...string) (*PodmanUnshareReport, error)
	UnshareWithCheck(check, image string, mounted bool) (*PodmanUnshareCheckReport, error)
}
