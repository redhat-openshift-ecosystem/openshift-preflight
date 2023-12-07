// Package version contains all identifiable versioning info for
// describing the preflight project.
package version

import (
	"context"
	"fmt"
	"strings"

	semver "github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v57/github"
	"github.com/spf13/cobra"
)

var (
	projectName = "github.com/redhat-openshift-ecosystem/openshift-preflight"
	version     = "unknown"
	commit      = "unknown"
)

var Version = VersionContext{
	Name:    projectName,
	Version: version,
	Commit:  commit,
}

type VersionClient interface {
	GetLatestRelease(ctx context.Context, owner string, repo string) (*github.RepositoryRelease, *github.Response, error)
}

type VersionContext struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

func (vc *VersionContext) String() string {
	return fmt.Sprintf("%s <commit: %s>", vc.Version, vc.Commit)
}

func (vc *VersionContext) LatestReleasedVersion(cmd *cobra.Command, svc VersionClient) (*github.RepositoryRelease, error) {
	ctx := cmd.Context()
	logger := logr.FromContextOrDiscard(ctx)

	projectTokens := strings.Split(vc.Name, "/")
	owner := projectTokens[1]
	repo := projectTokens[2]
	// Fetch latest release from Github
	latestRelease, resp, err := svc.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	logger.Info("Github responded with", "rate limit", resp.Rate.String())
	currentVersion, err := semver.NewVersion(vc.Version)
	if err != nil {
		logger.Error(err, "Unable to determine current semver")
		return nil, err
	}
	latestVersion, err := semver.NewVersion(*latestRelease.TagName)
	if err != nil {
		logger.Error(err, "Unable to determine latest semver")
		return nil, err
	}
	if !currentVersion.Equal(latestVersion) {
		return latestRelease, nil
	}
	return nil, nil
}
