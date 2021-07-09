package bindings

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/containers/podman/v3/pkg/bindings/containers"
	"github.com/containers/podman/v3/pkg/bindings/images"
	"github.com/containers/podman/v3/pkg/specgen"
)

func (p *PodmanClient) PullImage(nameOrID string) (*PullImageReport, error) {
	var quiet bool = true
	out, err := images.Pull(p.Context, nameOrID, &images.PullOptions{
		Quiet: &quiet,
	})
	if err != nil {
		return nil, err
	}
	return &PullImageReport{
		Output: strings.Join(out, "\n"),
		Tool:   reflect.TypeOf(p).String(),
	}, nil
}

func (p *PodmanClient) SaveImage(nameOrID string) (*SaveImageReport, error) {
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

	var compress bool = true
	err = images.Export(p.Context, []string{nameOrID}, outfile, &images.ExportOptions{
		Compress: &compress,
	})
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (p *PodmanClient) InspectImage(nameOrID string) (*InspectImageReport, error) {
	if _, err := p.PullImage(nameOrID); err != nil {
		return nil, err
	}
	report, err := images.GetImage(p.Context, nameOrID, &images.GetOptions{})
	if err != nil {
		return nil, err
	}

	return buildInspectImageReport(report), nil
}

func (p *PodmanClient) ListImages() (*ListImageReport, error) {
	list, err := images.List(p.Context, &images.ListOptions{})
	if err != nil {
		return nil, err
	}

	return &ListImageReport{
		Images: list,
	}, nil
}

func (p *PodmanClient) RemoveImage(nameOrID string) (*RemoveImageReport, error) {
	responses, err := images.Remove(p.Context, []string{nameOrID}, &images.RemoveOptions{})
	if err != nil {
		return nil, err[0]
	}

	return &RemoveImageReport{
		IDs: responses.Deleted,
	}, nil
}

func (p *PodmanClient) RunContainer(nameOrID string, options RunOptions) (*RunContainerReport, error) {
	if _, err := p.PullImage(nameOrID); err != nil {
		return nil, err
	}

	spec := specgen.NewSpecGenerator(nameOrID, false)
	if !reflect.DeepEqual(options, RunOptions{}) {
		spec.Terminal = options.Tty
		if len(options.Entrypoint) > 0 {
			spec.ContainerBasicConfig.Entrypoint = options.Entrypoint
		}

		if len(options.Cmd) > 0 {
			spec.ContainerBasicConfig.Command = options.Cmd
		}
	}
	container, err := containers.CreateWithSpec(p.Context, spec, &containers.CreateOptions{})
	if err != nil {
		return nil, err
	}

	if err := containers.Start(p.Context, container.ID, &containers.StartOptions{}); err != nil {
		return nil, err
	}

	return &RunContainerReport{
		ID: container.ID,
	}, nil
}

func (d *PodmanClient) ListContainers() (*ListContainerReport, error) {
	containerList, err := containers.List(d.Context, &containers.ListOptions{})
	if err != nil {
		return nil, err
	}

	return &ListContainerReport{
		Containers: containerList,
	}, nil
}

func (p *PodmanClient) RemoveContainer(nameOrID string) error {
	var forceDelete bool = true
	return containers.Remove(p.Context, nameOrID, &containers.RemoveOptions{
		Force: &forceDelete,
	})
}
