package bindings

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/containers/podman/v3/libpod/define"
	"github.com/containers/podman/v3/pkg/domain/entities"
	"github.com/containers/podman/v3/pkg/inspect"
	"github.com/docker/docker/api/types"
	"github.com/opencontainers/go-digest"
	specv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// getTarballName takes the name or ID of an image and
// returns the name of an archive
func getTarballName(nameOrID string) string {
	var imageURL string
	if strings.Contains(nameOrID, "@") {
		imageURL = strings.Split(nameOrID, "@")[0]
	}

	if strings.Contains(nameOrID, ":") {
		imageURL = strings.Split(nameOrID, ":")[0]
	}

	if imageURL == "" {
		imageURL = nameOrID
	}

	if strings.Contains(imageURL, "/") {
		imageName := strings.Split(imageURL, "/")
		return fmt.Sprintf("%s.tar.gz", imageName[len(imageName)-1])
	}

	return fmt.Sprintf("%s.tar.gz", imageURL)
}

// The image pull path defaults to /tmp
// when the IMAGE_PULL_PATH is not set
func imagePullPath() string {
	pullPath := "/tmp"
	overridePath, ok := os.LookupEnv("IMAGE_PULL_PATH")
	if ok && overridePath != "" {
		pullPath = overridePath
	}
	return pullPath
}

func parseTime(stringTime string) *time.Time {
	t, err := time.Parse(time.RFC3339, stringTime)
	if err != nil {
		return nil
	}
	return &t
}

func buildInspectImageReport(imageData interface{}) *InspectImageReport {
	switch reflect.TypeOf(imageData).String() {
	case "types.ImageInspect": // docker client
		return dockerInspectImageReport(imageData.(types.ImageInspect))
	case "*entities.ImageInspectReport": // podman client
		return podmanInspectImageReport(imageData.(*entities.ImageInspectReport))
	}

	return nil
}

// TODO: convert over exposed ports: report.ContainerConfig.ExposedPorts
func dockerInspectImageReport(report types.ImageInspect) *InspectImageReport {
	var layerDigests []digest.Digest
	for _, layer := range report.RootFS.Layers {
		layerDigests = append(layerDigests, digest.Digest(layer))
	}

	return &InspectImageReport{
		ImageData: &inspect.ImageData{
			ID:          report.ID,
			RepoTags:    report.RepoTags,
			RepoDigests: report.RepoDigests,
			Created:     parseTime(report.Created),
			Parent:      report.Parent,
			Comment:     report.Comment,
			GraphDriver: &define.DriverData{
				Name: report.GraphDriver.Name,
				Data: report.GraphDriver.Data,
			},
			Os:           report.Os,
			Size:         report.Size,
			VirtualSize:  report.VirtualSize,
			Author:       report.Author,
			Version:      report.DockerVersion,
			Architecture: report.Architecture,
			RootFS: &inspect.RootFS{
				Type:   report.RootFS.Type,
				Layers: layerDigests,
			},
			Config: &specv1.ImageConfig{
				User:       report.ContainerConfig.User,
				Env:        report.ContainerConfig.Env,
				Entrypoint: report.ContainerConfig.Entrypoint,
				Cmd:        report.ContainerConfig.Cmd,
				Volumes:    report.ContainerConfig.Volumes,
				WorkingDir: report.ContainerConfig.WorkingDir,
				Labels:     report.ContainerConfig.Labels,
				StopSignal: report.ContainerConfig.StopSignal,
				// ExposedPorts: report.ContainerConfig.ExposedPorts,
			},
		},
	}

}

func podmanInspectImageReport(report *entities.ImageInspectReport) *InspectImageReport {
	return &InspectImageReport{
		ImageData: report.ImageData,
	}
}

// convert docker list of images into ListImageReport
func dockerListImageReport(images []types.ImageSummary) *ListImageReport {
	var imageList []*entities.ImageSummary
	for _, image := range images {
		imageList = append(imageList, &entities.ImageSummary{
			ID:          image.ID,
			ParentId:    image.ParentID,
			Created:     image.Created,
			RepoTags:    image.RepoTags,
			RepoDigests: image.RepoDigests,
			Labels:      image.Labels,
			Containers:  int(image.Containers),
			Size:        image.Size,
			SharedSize:  int(image.SharedSize),
			VirtualSize: image.VirtualSize,
		})
	}
	return &ListImageReport{
		Images: imageList,
	}
}

// TODO: add coverage for the data that is converted/returned
func dockerListContainerReport(containers []types.Container) *ListContainerReport {
	var containerList []entities.ListContainer
	for _, container := range containers {
		containerList = append(containerList, entities.ListContainer{
			ID:      container.ID,
			Command: strings.Split(container.Command, " "),
			Created: time.Unix(container.Created, 0),
			Image:   container.Image,
			ImageID: container.ImageID,
			Labels:  container.Labels,
			Names:   container.Names,
		})
	}
	return &ListContainerReport{
		Containers: containerList,
	}
}
