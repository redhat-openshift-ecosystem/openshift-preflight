package bindings

import (
	"bytes"
	"path/filepath"
	"reflect"

	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

func (d *DockerClient) PullImage(nameOrID string) (*PullImageReport, error) {
	reader, err := d.Client.ImagePull(d.Context, nameOrID, types.ImagePullOptions{})
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	buffer := new(bytes.Buffer)
	buffer.ReadFrom(reader)

	return &PullImageReport{
		Output: buffer.String(),
		Tool:   reflect.TypeOf(d).String(),
	}, nil
}

// TODO: allow user to change directory to store archives via env variable.
// Also, add "/tmp" as default image path
func (d *DockerClient) SaveImage(nameOrID string) (*SaveImageReport, error) {
	response := &SaveImageReport{
		Filename:  getTarballName(nameOrID),
		Directory: imagePullPath(),
	}
	response.AbsPath = filepath.Join(response.Directory, response.Filename)
	outfile, err := os.Create(response.AbsPath)
	if err != nil {
		return nil, err
	}
	defer outfile.Close()

	reader, err := d.Client.ImageSave(d.Context, []string{nameOrID})
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	outfile.ReadFrom(reader)

	return response, nil
}

func (d *DockerClient) InspectImage(nameOrID string) (*InspectImageReport, error) {
	if _, err := d.PullImage(nameOrID); err != nil {
		return nil, err
	}
	report, _, err := d.Client.ImageInspectWithRaw(d.Context, nameOrID)
	if err != nil {
		return nil, err
	}

	return buildInspectImageReport(report), nil
}

func (d *DockerClient) ListImages() (*ListImageReport, error) {
	list, err := d.Client.ImageList(d.Context, types.ImageListOptions{})
	if err != nil {
		return nil, err
	}
	images := dockerListImageReport(list)

	return images, nil
}

func (d *DockerClient) RemoveImage(nameOrID string) (*RemoveImageReport, error) {
	responses, err := d.Client.ImageRemove(d.Context, nameOrID, types.ImageRemoveOptions{})
	if err != nil {
		return nil, err
	}

	var deleted []string
	for _, response := range responses {
		deleted = append(deleted, response.Deleted)
	}

	return &RemoveImageReport{
		IDs: deleted,
	}, nil
}

func (d *DockerClient) RunContainer(nameOrID string, options RunOptions) (*RunContainerReport, error) {
	if _, err := d.PullImage(nameOrID); err != nil {
		return nil, err
	}

	spec := &container.Config{
		Image: nameOrID,
	}

	if !reflect.DeepEqual(options, RunOptions{}) {
		spec.Tty = options.Tty
		if len(options.Entrypoint) > 0 {
			spec.Entrypoint = options.Entrypoint
		}

		if len(options.Cmd) > 0 {
			spec.Cmd = options.Cmd
		}
	}

	containerCreated, err := d.Client.ContainerCreate(d.Context, spec, nil, nil, nil, "")
	if err != nil {
		return nil, err
	}

	if err := d.Client.ContainerStart(d.Context, containerCreated.ID, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}

	return &RunContainerReport{
		ID: containerCreated.ID,
	}, nil
}

func (d *DockerClient) ListContainers() (*ListContainerReport, error) {
	list, err := d.Client.ContainerList(d.Context, types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}
	containerListReport := dockerListContainerReport(list)
	return containerListReport, nil
}

func (d *DockerClient) RemoveContainer(nameOrID string) error {
	return d.Client.ContainerRemove(d.Context, nameOrID, types.ContainerRemoveOptions{
		Force: true,
	})
}
