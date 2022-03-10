package container

import (
	"fmt"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/service"
)

func NewHasUniqueTagCheck(tagLister service.TagLister) *HasUniqueTagCheck {
	return &HasUniqueTagCheck{TagLister: tagLister}
}

// HasUniqueTagCheck evaluates the image to ensure that it has a tag other than
// the latest tag, which is considered to be a "floating" tag and may not accurately
// represent the same image over time.
type HasUniqueTagCheck struct {
	TagLister service.TagLister
}

func (p *HasUniqueTagCheck) Validate(imgRef certification.ImageReference) (bool, error) {
	tags, err := p.getDataToValidate(fmt.Sprintf("%s/%s", imgRef.ImageRegistry, imgRef.ImageRepository))
	if err != nil {
		return false, err
	}
	return p.validate(tags)
}

func (p *HasUniqueTagCheck) getDataToValidate(image string) ([]string, error) {
	return p.TagLister.ListTags(image)
}

func (p *HasUniqueTagCheck) validate(tags []string) (bool, error) {
	// An image passes the check if:
	// 1) it has more than one tag (`latest` is acceptable)
	// OR
	// 2) it has only one tag, and it is not `latest`
	return len(tags) > 1 || len(tags) == 1 && strings.ToLower(tags[0]) != "latest", nil
}

func (p *HasUniqueTagCheck) Name() string {
	return "HasUniqueTag"
}

func (p *HasUniqueTagCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if container has a tag other than 'latest', so that the image can be uniquely identified.",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *HasUniqueTagCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check HasUniqueTag encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Add a tag to your image. Consider using Semantic Versioning. https://semver.org/",
	}
}
