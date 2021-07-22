package bindings

import (
	"fmt"
	"os"
	"strconv"
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
		return fmt.Sprintf("%s.tar", imageName[len(imageName)-1])
	}

	return fmt.Sprintf("%s.tar", imageURL)
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

// the dockerInspectImageReport  converts the data returned by docker
// image inspect and returns a pointer reference to InspectImageReport
// TODO: convert over exposed ports: report.ContainerConfig.ExposedPorts
func dockerInspectImageReport(report types.ImageInspect) *InspectImageReport {
	var layerDigests []digest.Digest
	for _, layer := range report.RootFS.Layers {
		layerDigests = append(layerDigests, digest.Digest(layer))
	}

	exposedPorts := make(map[string]struct{})
	for name, portset := range report.ContainerConfig.ExposedPorts {
		exposedPorts[string(name)] = portset
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
				User:         report.ContainerConfig.User,
				Env:          report.ContainerConfig.Env,
				Entrypoint:   report.ContainerConfig.Entrypoint,
				Cmd:          report.ContainerConfig.Cmd,
				Volumes:      report.ContainerConfig.Volumes,
				WorkingDir:   report.ContainerConfig.WorkingDir,
				Labels:       report.ContainerConfig.Labels,
				StopSignal:   report.ContainerConfig.StopSignal,
				ExposedPorts: exposedPorts,
			},
		},
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

// dockerInspectContainerReport takes the output provided by docker container inspect
// command and returns a pointer to InspectContainerReport
// TODO: fix container.HostConfig.Healthcheck
func dockerInspectContainerReport(container types.ContainerJSON) *InspectContainerReport {
	onBuild := strings.Join(container.Config.OnBuild, " ")
	stopSignal, _ := strconv.ParseUint(container.Config.StopSignal, 10, 32)

	report := &InspectContainerReport{
		InspectContainerData: &define.InspectContainerData{
			ID:              container.ID,
			Created:         *parseTime(container.Created),
			Path:            container.Path,
			Args:            container.Args,
			State:           &define.InspectContainerState{},
			Image:           container.Image,
			ImageName:       container.Image,
			ResolvConfPath:  container.ResolvConfPath,
			HostnamePath:    container.HostnamePath,
			HostsPath:       container.HostsPath,
			Name:            container.Name,
			RestartCount:    int32(container.RestartCount),
			Driver:          container.Driver,
			MountLabel:      container.MountLabel,
			ProcessLabel:    container.ProcessLabel,
			AppArmorProfile: container.AppArmorProfile,
			ExecIDs:         container.ExecIDs,
			GraphDriver: &define.DriverData{
				Name: container.GraphDriver.Name,
				Data: container.GraphDriver.Data,
			},
			SizeRw:          container.SizeRw,
			SizeRootFs:      *container.SizeRootFs,
			Mounts:          []define.InspectMount{},
			NetworkSettings: &define.InspectNetworkSettings{},
			Config: &define.InspectContainerConfig{
				Hostname:     container.Config.Hostname,
				DomainName:   container.Config.Domainname,
				User:         container.Config.User,
				AttachStdin:  container.Config.AttachStdin,
				AttachStdout: container.Config.AttachStdout,
				Tty:          container.Config.Tty,
				OpenStdin:    container.Config.OpenStdin,
				StdinOnce:    container.Config.StdinOnce,
				Env:          container.Config.Env,
				Cmd:          container.Config.Cmd,
				// Healthcheck: &manifest.Schema2HealthConfig{
				// 	Test:        containerConfig.Healthcheck.Test,
				// 	StartPeriod: containerConfig.Healthcheck.StartPeriod,
				// 	Interval:    containerConfig.Healthcheck.Interval,
				// 	Timeout:     containerConfig.Healthcheck.Timeout,
				// 	Retries:     containerConfig.Healthcheck.Retries,
				// },
				Image:       container.Config.Image,
				Volumes:     container.Config.Volumes,
				WorkingDir:  container.Config.WorkingDir,
				Entrypoint:  strings.Join(container.Config.Entrypoint, " "),
				OnBuild:     &onBuild,
				Labels:      container.Config.Labels,
				StopSignal:  uint(stopSignal),
				StopTimeout: uint(*container.Config.StopTimeout),
			},
			HostConfig: &define.InspectContainerHostConfig{
				Binds:             container.HostConfig.Binds,
				ContainerIDFile:   container.HostConfig.ContainerIDFile,
				NetworkMode:       string(container.HostConfig.NetworkMode),
				PortBindings:      map[string][]define.InspectHostPort{},
				AutoRemove:        container.HostConfig.AutoRemove,
				VolumeDriver:      container.HostConfig.VolumeDriver,
				VolumesFrom:       container.HostConfig.VolumesFrom,
				CapAdd:            container.HostConfig.CapAdd,
				CapDrop:           container.HostConfig.CapDrop,
				Dns:               container.HostConfig.DNS,
				DnsOptions:        container.HostConfig.DNSOptions,
				DnsSearch:         container.HostConfig.DNSSearch,
				ExtraHosts:        container.HostConfig.ExtraHosts,
				GroupAdd:          container.HostConfig.GroupAdd,
				IpcMode:           string(container.HostConfig.IpcMode),
				Cgroup:            string(container.HostConfig.Cgroup),
				CgroupMode:        string(container.HostConfig.CgroupnsMode),
				Links:             container.HostConfig.Links,
				OomScoreAdj:       container.HostConfig.OomScoreAdj,
				PidMode:           string(container.HostConfig.PidMode),
				Privileged:        container.HostConfig.Privileged,
				PublishAllPorts:   container.HostConfig.PublishAllPorts,
				ReadonlyRootfs:    container.HostConfig.ReadonlyRootfs,
				SecurityOpt:       container.HostConfig.SecurityOpt,
				Tmpfs:             container.HostConfig.Tmpfs,
				UTSMode:           string(container.HostConfig.UTSMode),
				UsernsMode:        string(container.HostConfig.UsernsMode),
				ShmSize:           container.HostConfig.ShmSize,
				Runtime:           container.HostConfig.Runtime,
				ConsoleSize:       container.HostConfig.ConsoleSize[:],
				Isolation:         string(container.HostConfig.Isolation),
				CpuShares:         uint64(container.HostConfig.CPUShares),
				Memory:            container.HostConfig.Memory,
				NanoCpus:          container.HostConfig.NanoCPUs,
				CgroupParent:      container.HostConfig.CgroupParent,
				BlkioWeight:       container.HostConfig.BlkioWeight,
				BlkioWeightDevice: []define.InspectBlkioWeightDevice{},
				LogConfig: &define.InspectLogConfig{
					Type:   container.HostConfig.LogConfig.Type,
					Config: container.HostConfig.LogConfig.Config,
				},
				RestartPolicy: &define.InspectRestartPolicy{
					Name:              container.HostConfig.RestartPolicy.Name,
					MaximumRetryCount: uint(container.HostConfig.RestartPolicy.MaximumRetryCount),
				},
				Init: *container.HostConfig.Init,
			},
		},
	}

	for _, mount := range container.Mounts {
		report.Mounts = append(report.Mounts, define.InspectMount{
			Type:        string(mount.Type),
			Name:        mount.Name,
			Source:      mount.Source,
			Destination: mount.Destination,
			Driver:      mount.Driver,
			Mode:        mount.Mode,
			RW:          mount.RW,
		})
	}

	for _, blkioWeightDev := range container.HostConfig.BlkioWeightDevice {
		report.HostConfig.BlkioWeightDevice = append(report.HostConfig.BlkioWeightDevice,
			define.InspectBlkioWeightDevice{
				Path:   blkioWeightDev.Path,
				Weight: blkioWeightDev.Weight,
			})
	}

	for name, portBindings := range container.HostConfig.PortBindings {
		bindings := []define.InspectHostPort{}
		for _, bind := range portBindings {
			bindings = append(bindings, define.InspectHostPort{
				HostIP:   bind.HostIP,
				HostPort: bind.HostPort,
			})
		}

		report.HostConfig.PortBindings[string(name)] = bindings
	}

	return report
}
