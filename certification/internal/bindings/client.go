package bindings

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containers/podman/v3/pkg/bindings"
	"github.com/docker/docker/client"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
)

func podmanSocket() string {
	socketPath := []string{
		os.Getenv("XDG_RUNTIME_DIR"),
		"podman",
		"podman.sock",
	}
	if _, err := os.Stat(filepath.Join(socketPath...)); err != nil {
		socketPath = []string{
			"/run",
			"podman",
			"podman.sock",
		}
	}

	return fmt.Sprintf("unix://%s", filepath.Join(socketPath...))
}

// if podman socket is not found, create a socket for the docker client
// Should docker not be running, the function should return an error
// indicating neither podman or docker socket could be found
func Client() (ContainerTool, error) {
	ctx, err := bindings.NewConnection(context.Background(), podmanSocket())
	if err != nil {
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			return nil, errors.ErrNoSocketFound
		}
		return &DockerClient{
			Client:  cli,
			Context: context.Background(),
		}, nil

	}
	return &PodmanClient{
		Context: ctx,
	}, nil
}
