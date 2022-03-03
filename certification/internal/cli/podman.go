package cli

import "time"

type PodmanCreateOutput struct {
	ContainerId string
	Stdout      string
	Stderr      string
}

type PodmanOutput struct {
	Stdout string
	Stderr string
}

type PodmanCreateOption struct {
	Entrypoint    []string
	Cmd           []string
	ContainerName string
}

type ImagePullOptions struct {
	Quiet bool
}

type WaitOptions struct {
	Interval  string
	Condition string
	Timeout   time.Duration
}

type InspectContainerData struct {
	ID     string                 `json:"Id"`
	Config InspectContainerConfig `json:"Config"`
}

type InspectContainerConfig struct {
	Cmd        []string `json:"Cmd"`
	Entrypoint string   `json:"Entrypoint"`
}

type PodmanEngine interface {
	CreateContainer(imageURI string, createOptions PodmanCreateOption) (*PodmanCreateOutput, error)
	StartContainer(nameOrId string) (*PodmanOutput, error)
	RemoveContainer(containerId string) error
	WaitContainer(containerId string, waitOptions WaitOptions) (bool, error)
	RunSystemContainer(containerName string) (*PodmanOutput, error)
	IsSystemContainerRunning(serviceName string) (bool, error)
	StopSystemContainer(serviceName string) error
}
