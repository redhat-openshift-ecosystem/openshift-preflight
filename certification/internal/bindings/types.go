package bindings

import (
	"context"

	"github.com/containers/podman/v3/pkg/domain/entities"
	"github.com/containers/podman/v3/pkg/inspect"
	"github.com/docker/docker/client"
)

// PodmanClient is a client used to interact with
// containers and images via the with podman socket
type PodmanClient struct {
	context.Context
}

// DockerSocket is a client used to interact with
// containers and images via the with podman socket
type DockerClient struct {
	*client.Client
	context.Context
}

// Container struct provides a way to standardize the results returned
// by the methods run by the podman client and docker client methods
type Container struct {
	ID     string
	Names  []string
	Labels map[string]string
}

// Image struct provides a way to standardize the results returned
// by the methods run by the podman client and docker client methods
type Image struct {
	ID          string
	Labels      map[string]string
	RepoTags    []string
	RepoDigests []string
}

// The ContainerTool interface provides a unified way to interact with
// both the podman and docker methods below
type ContainerTool interface {
	// PullImage takes an image name and pulls the image to the local cache
	PullImage(nameOrID string) (*PullImageReport, error)
	// InspectImage returns details on an image in the local cache
	InspectImage(nameOrID string) (*InspectImageReport, error)
	// SaveImage saves an image in a tarball on the local filesystem
	SaveImage(nameOrID string) (*SaveImageReport, error)
	// ListImages returns a list of images available on the local cache
	ListImages() (*ListImageReport, error)
	// RemoveImage deletes an image from the local cache
	RemoveImage(nameOrID string) (*RemoveImageReport, error)
	// RunContainer creates and starts a container
	RunContainer(nameOrID string, options RunOptions) (*RunContainerReport, error)
	// ListContainer lists the containers on the local system
	ListContainers() (*ListContainerReport, error)
	// RemoveContainer deletes a container running on the local system
	RemoveContainer(nameOrID string) error
}

type PullImageReport struct {
	Output string
	Tool   string
}

type InspectImageReport struct {
	*inspect.ImageData
}

type SaveImageReport struct {
	Filename  string
	Directory string
	AbsPath   string
}

type ListImageReport struct {
	Images []*entities.ImageSummary
}

type RemoveImageReport struct {
	IDs []string
}

type RunContainerReport struct {
	// ID of the container that was started
	ID string
}

type ListContainerReport struct {
	Containers []entities.ListContainer
}

type RunOptions struct {
	Entrypoint []string
	Cmd        []string
	Tty        bool
}
