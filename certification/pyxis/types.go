package pyxis

import "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"

type CertImage struct {
	ID                     string       `json:"_id,omitempty"`
	Certified              bool         `json:"certified"`
	Deleted                bool         `json:"deleted" default:"false"`
	DockerImageDigest      string       `json:"docker_image_digest,omitempty"`
	DockerImageID          string       `json:"docker_image_id,omitempty"`
	ImageID                string       `json:"image_id,omitempty"`
	ISVPID                 string       `json:"isv_pid,omitempty"` // required
	ParsedData             *ParsedData  `json:"parsed_data,omitempty"`
	Architecture           string       `json:"architecture" default:"amd64"`
	RawConfig              string       `json:"raw_config,omitempty"`
	Repositories           []Repository `json:"repositories,omitempty"`
	SumLayerSizeBytes      int64        `json:"sum_layer_size_bytes,omitempty"`
	UncompressedTopLayerId string       `json:"uncompressed_top_layer_id,omitempty"` // TODO: figure out how to populate this, it is not required
}

type ParsedData struct {
	Architecture           string  `json:"architecture,omitempty"`
	Command                string  `json:"command,omitempty"`
	Comment                string  `json:"comment,omitempty"`
	Container              string  `json:"container,omitempty"`
	Created                string  `json:"created,omitempty"`
	DockerVersion          string  `json:"docker_version,omitempty"`
	ImageID                string  `json:"image_id,omitempty"`
	Labels                 []Label `json:"labels,omitempty"` // required
	OS                     string  `json:"os,omitempty"`
	Ports                  string  `json:"ports,omitempty"`
	Size                   int64   `json:"size,omitempty"`
	UncompressedLayerSizes []Layer `json:"uncompressed_layer_sizes,omitempty"`
}

type Repository struct {
	Published  bool   `json:"published" default:"false"`
	PushDate   string `json:"push_date,omitempty"` // time.Now
	Registry   string `json:"registry,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tags       []Tag  `json:"tags,omitempty"`
}

type Label struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Tag struct {
	AddedDate string `json:"added_date,omitempty"` // time.Now
	Name      string `json:"name,omitempty"`
}

type RPMManifest struct {
	ID      string `json:"_id,omitempty"`
	ImageID string `json:"image_id,omitempty"`
	RPMS    []RPM  `json:"rpms,omitempty"`
}

type RPM struct {
	Architecture string `json:"architecture,omitempty"`
	Gpg          string `json:"gpg,omitempty"`
	Name         string `json:"name,omitempty"`
	Nvra         string `json:"nvra,omitempty"`
	Release      string `json:"release,omitempty"`
	SrpmName     string `json:"srpm_name,omitempty"`
	SrpmNevra    string `json:"srpm_nevra,omitempty"`
	Summary      string `json:"summary,omitempty"`
	Version      string `json:"version,omitempty"`
}

type CertProject struct {
	ID                  string    `json:"_id,omitempty"`
	CertificationStatus string    `json:"certification_status" default:"In Progress"`
	Container           Container `json:"container"`
	Name                string    `json:"name"`                      // required
	ProjectStatus       string    `json:"project_status"`            // required
	Type                string    `json:"type" default:"Containers"` // required
	OsContentType       string    `json:"os_content_type,omitempty"`
}

type Container struct {
	DockerConfigJSON string `json:"docker_config_json"`
	Type             string `json:"type" default:"Containers"` // conditionally required
	ISVPID           string `json:"isv_pid,omitempty"`         // required
}

type Layer struct {
	LayerId string `json:"layer_id"`
	Size    int64  `json:"size_bytes"`
}

type TestResults struct {
	ID          string `json:"_id,omitempty"`
	CertProject string `json:"cert_project,omitempty"`
	OrgID       int    `json:"org_id,omitempty"`
	Version     string `json:"version,omitempty"`
	ImageID     string `json:"image_id,omitempty"`
	formatters.UserResponse
}
