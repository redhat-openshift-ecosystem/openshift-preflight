package pyxis

//todo-adam reduce these fields to just what we need for now
type CertImage struct {
	Certified              bool          `json:"certified" default:"false"` //todo-adam I'm going to assume we want to set this to false??
	Deleted                bool          `json:"deleted" default:"false"`
	DockerImageDigest      string        `json:"docker_image_digest,omitempty"`
	DockerImageID          string        `json:"docker_image_id,omitempty"`
	ImageID                string        `json:"image_id,omitempty"`
	ISVPID                 string        `json:"isv_pid,omitempty"`
	ParsedData             *ParsedData   `json:"parsed_data,omitempty"`
	RawConfig              string        `json:"raw_config,omitempty"`
	Repositories           *Repositories `json:"repositories,omitempty"`
	SumLayerSizeBytes      int32         `json:"sum_layer_size_bytes,omitempty"`
	UncompressedTopLayerId string        `json:"uncompressed_top_layer_id,omitempty"`
}

type ParsedData struct {
	Architecture           string  `json:"architecture,omitempty"`
	Command                string  `json:"command,omitempty"`
	Comment                string  `json:"comment,omitempty"`
	Container              string  `json:"container,omitempty"`
	DockerVersion          string  `json:"docker_version,omitempty"`
	FilesNum               int32   `json:"files#"`
	ImageID                string  `json:"image_id,omitempty"`
	OS                     string  `json:"os,omitempty"`
	Ports                  string  `json:"ports,omitempty"`
	Size                   int32   `json:"size,omitempty"`
	UncompressedLayerSizes []Layer `json:"uncompressed_layer_sizes,omitempty"`
}

type Repositories []struct {
	Published  bool   `json:"published" default:"false"`
	PushDate   string `json:"push_date,omitempty"` // time.Now
	Registry   string `json:"registry,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tags       *Tags  `json:"tags,omitempty"`
}

type Tags []struct {
	AddedDate string `json:"added_date,omitempty"` // time.Now
	Name      string `json:"name,omitempty"`
}

type RPMManifest struct {
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

//todo-adam make sure to set certProject.certification_status to "In Progress"
type CertProject struct {
	CertificationStatus string    `json:"certification_status" default:"In Progress"`
	Container           Container `json:"container"`
	Name                string    `json:"name"`                      // required
	ProjectStatus       string    `json:"project_status"`            // required
	Type                string    `json:"type" default:"Containers"` // required
	OsContentType       string    `json:"os_content_type,omitempty"`
}

type Container struct {
	DockerConfigJSON string `json:"docker_config_json"`
	Type             string `json:"type " default:"Containers"` // conditionally required
}

type Layer struct {
	LayerId string `json:"layer_id"`
	Size    uint64 `json:"size_bytes"`
}
