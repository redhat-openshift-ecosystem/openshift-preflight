package container

import (
	"context"
	"fmt"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/authn"

	"github.com/google/go-containerregistry/pkg/crane"
)

var _ certification.Check = &hasUniqueTagCheck{}

func NewHasUniqueTagCheck(dockercfg string) *hasUniqueTagCheck {
	return &hasUniqueTagCheck{
		dockercfg: dockercfg,
	}
}

// HasUniqueTagCheck evaluates the image to ensure that it has a tag other than
// the latest tag, which is considered to be a "floating" tag and may not accurately
// represent the same image over time.
type hasUniqueTagCheck struct {
	dockercfg string
}

func (p *hasUniqueTagCheck) Validate(ctx context.Context, imgRef certification.ImageReference) (bool, error) {
	imgRepo := fmt.Sprintf("%s/%s", imgRef.ImageRegistry, imgRef.ImageRepository)

	tags := make([]string, 0)
	var err error
	// if sha or latest tag is passed in `/tags/list` must be exposed and available to validate that the image is being tagged properly
	if strings.HasPrefix(imgRef.ImageTagOrSha, "sha256:") || imgRef.ImageTagOrSha == "latest" {
		tags, err = p.getDataToValidate(ctx, imgRepo)
		if err != nil {
			return false, fmt.Errorf("failed to get tags list for %s: %v", imgRepo, err)
		}
	}

	// if tags is of length zero we know that either
	// the partners registry returned an empty list so fall back
	// or the value imgRef.ImageTagOrSha did not meet the previous conditions so falling back to use the value passed in
	if len(tags) == 0 {
		if strings.HasPrefix(imgRef.ImageTagOrSha, "sha256:") {
			return false, fmt.Errorf("no tags found for %s: cannot assert tag from digest", imgRepo)
		}

		tags = append(tags, imgRef.ImageTagOrSha)
	}

	return p.validate(tags)
}

func (p *hasUniqueTagCheck) getDataToValidate(ctx context.Context, image string) ([]string, error) {
	options := []crane.Option{
		crane.WithContext(ctx),
		crane.WithAuthFromKeychain(authn.PreflightKeychain(authn.WithDockerConfig(p.dockercfg))),
	}

	return crane.ListTags(image, options...)
}

func (p *hasUniqueTagCheck) validate(tags []string) (bool, error) {
	// An image passes the check if:
	// 1) it has more than one tag (`latest` is acceptable)
	// OR
	// 2) it has only one tag, and it is not `latest`
	return len(tags) > 1 || len(tags) == 1 && strings.ToLower(tags[0]) != "latest", nil
}

func (p *hasUniqueTagCheck) Name() string {
	return "HasUniqueTag"
}

func (p *hasUniqueTagCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if container has a tag other than 'latest', so that the image can be uniquely identified.",
		Level:            "best",
		KnowledgeBaseURL: certDocumentationURL,
		CheckURL:         certDocumentationURL,
	}
}

func (p *hasUniqueTagCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check HasUniqueTag encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Add a tag to your image. Consider using Semantic Versioning. https://semver.org/",
	}
}
