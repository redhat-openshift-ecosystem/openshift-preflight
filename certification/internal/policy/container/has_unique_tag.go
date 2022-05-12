package container

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/authn"
)

func NewHasUniqueTagCheck() *hasUniqueTagCheck {
	return &hasUniqueTagCheck{}
}

// HasUniqueTagCheck evaluates the image to ensure that it has a tag other than
// the latest tag, which is considered to be a "floating" tag and may not accurately
// represent the same image over time.
type hasUniqueTagCheck struct{}

func (p *hasUniqueTagCheck) Validate(ctx context.Context, imgRef certification.ImageReference) (bool, error) {
	tags, err := p.getDataToValidate(ctx, fmt.Sprintf("%s/%s", imgRef.ImageRegistry, imgRef.ImageRepository))
	if err != nil {
		return false, err
	}
	return p.validate(tags)
}

func (p *hasUniqueTagCheck) getDataToValidate(ctx context.Context, image string) ([]string, error) {
	options := []crane.Option{
		crane.WithContext(ctx),
		crane.WithAuthFromKeychain(authn.PreflightKeychain),
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
