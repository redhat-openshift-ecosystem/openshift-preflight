package container

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/authn"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

var _ check.Check = &hasUniqueTagCheck{}

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

func (p *hasUniqueTagCheck) Validate(ctx context.Context, imgRef image.ImageReference) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)
	imgRepo := fmt.Sprintf("%s/%s", imgRef.ImageRegistry, imgRef.ImageRepository)

	tags := make([]string, 0)
	var err error
	// if sha or latest tag is passed in `/tags/list` must be exposed and available to validate that the image is being tagged properly
	if strings.HasPrefix(imgRef.ImageTagOrSha, "sha256:") || imgRef.ImageTagOrSha == "latest" {
		tags, err = p.getDataToValidate(ctx, imgRepo)
		if err != nil {
			return false, fmt.Errorf("failed to get tags list for %s: %v", imgRepo, err)
		}

		logger.V(log.DBG).WithValues("tags", tags).Info(fmt.Sprintf("got tags list for %s", imgRepo))
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
	repo, err := name.NewRepository(image)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image name: %v", err)
	}

	options := []remote.Option{
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(authn.PreflightKeychain(ctx, authn.WithDockerConfig(p.dockercfg))),

		// A smaller query is fine for evaluating the HasUniqueTag check
		// https://github.com/redhat-openshift-ecosystem/openshift-preflight/pull/1268#discussion_r2085387116
		remote.WithPageSize(10),

		remote.WithRetryBackoff(remote.Backoff{
			Duration: 5 * time.Second,
			Factor:   1.0,
			Jitter:   0.1,
			Steps:    2,
		}),
	}

	puller, err := remote.NewPuller(options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create puller: %v", err)
	}

	lister, err := puller.Lister(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create lister: %v", err)
	}

	if lister.HasNext() {
		tags, err := lister.Next(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get tags: %v", err)
		}

		if tags != nil {
			return tags.Tags, nil
		}
	}

	return nil, nil
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

func (p *hasUniqueTagCheck) Metadata() check.Metadata {
	return check.Metadata{
		Description:      "Checking if container has a tag other than 'latest', so that the image can be uniquely identified.",
		Level:            "best",
		KnowledgeBaseURL: certDocumentationURL,
		CheckURL:         certDocumentationURL,
	}
}

func (p *hasUniqueTagCheck) Help() check.HelpText {
	return check.HelpText{
		Message:    "Check HasUniqueTag encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Add a tag to your image. Consider using Semantic Versioning. https://semver.org/",
	}
}

func (p *hasUniqueTagCheck) RequiredFilePatterns() []string {
	return nil
}
