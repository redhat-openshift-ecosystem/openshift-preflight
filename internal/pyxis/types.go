package pyxis

import (
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/formatters"
)

type CertificationInput struct {
	CertProject *CertProject
	CertImage   *CertImage
	TestResults *TestResults
	RpmManifest *RPMManifest
	Artifacts   []Artifact
}

type CertificationResults struct {
	CertProject *CertProject
	CertImage   *CertImage
	TestResults *TestResults
}

type CertImage struct {
	ID                     string           `json:"_id,omitempty"`
	Certified              bool             `json:"certified"`
	Deleted                bool             `json:"deleted" default:"false"`
	DockerImageDigest      string           `json:"docker_image_digest,omitempty"`
	DockerImageID          string           `json:"docker_image_id,omitempty"`
	ImageID                string           `json:"image_id,omitempty"`
	ISVPID                 string           `json:"isv_pid,omitempty"` // required
	ParsedData             *ParsedData      `json:"parsed_data,omitempty"`
	Architecture           string           `json:"architecture" default:"amd64"`
	RawConfig              string           `json:"raw_config,omitempty"`
	Repositories           []Repository     `json:"repositories,omitempty"`
	SumLayerSizeBytes      int64            `json:"sum_layer_size_bytes,omitempty"`
	UncompressedTopLayerID string           `json:"uncompressed_top_layer_id,omitempty"`
	FreshnessGrades        []FreshnessGrade `json:"freshness_grades,omitempty"`
}

type FreshnessGrade struct {
	Grade     string
	StartDate time.Time
	EndDate   time.Time
}

type ParsedData struct {
	Architecture           string   `json:"architecture,omitempty"`
	Command                string   `json:"command,omitempty"`
	Comment                string   `json:"comment,omitempty"`
	Container              string   `json:"container,omitempty"`
	Created                string   `json:"created,omitempty"`
	DockerVersion          string   `json:"docker_version,omitempty"`
	ImageID                string   `json:"image_id,omitempty"`
	Labels                 []Label  `json:"labels,omitempty"` // required
	Layers                 []string `json:"layers,omitempty"` // required
	OS                     string   `json:"os,omitempty"`
	Ports                  string   `json:"ports,omitempty"`
	Size                   int64    `json:"size,omitempty"`
	UncompressedLayerSizes []Layer  `json:"uncompressed_layer_sizes,omitempty"`
}

type Repository struct {
	Published          bool   `json:"published" default:"false"`
	PushDate           string `json:"push_date,omitempty"` // time.Now
	Registry           string `json:"registry,omitempty"`
	Repository         string `json:"repository,omitempty"`
	Tags               []Tag  `json:"tags,omitempty"`
	ManifestListDigest string `json:"manifest_list_digest,omitempty"`
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
	Name                string    `json:"name"`           // required
	ProjectStatus       string    `json:"project_status"` // required
	Type                string    `json:"type,omitempty"` // required
}

func (cp CertProject) ScratchProject() bool {
	// ScratchProject returns true if the CertProject is designated Scratch in Pyxis.
	return cp.Container.Type == "scratch" || cp.Container.OsContentType == "Scratch Image"
}

type Container struct {
	DockerConfigJSON string `json:"docker_config_json,omitempty"`
	HostedRegistry   bool   `json:"hosted_registry,omitempty"`
	Type             string `json:"type,omitempty"`    // conditionally required
	ISVPID           string `json:"isv_pid,omitempty"` // required
	Registry         string `json:"registry,omitempty"`
	Repository       string `json:"repository,omitempty"`
	OsContentType    string `json:"os_content_type,omitempty"`
	Privileged       bool   `json:"privileged,omitempty"`
}

type Layer struct {
	LayerID string `json:"layer_id"`
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

type Artifact struct {
	ID          string `json:"_id"`
	CertProject string `json:"cert_project"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	FileSize    int64  `json:"file_size"`
	Filename    string `json:"filename"`
	ImageID     string `json:"image_id"`
}
