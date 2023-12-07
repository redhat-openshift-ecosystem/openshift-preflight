package version

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/bombsimon/logrusr/v4"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v57/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/spf13/cobra"
)

var _ = Describe("version package utility", func() {
	// Values assumed to be passed when calling make test.
	ldflagVersionOverride := "0.0.1"
	ldflagCommitOverride := "foobar"

	// These tests validate that we can override the version and commit information successfully,
	// and that our string representation includes that information.
	Context("When being passed version and commit information via ldflags", func() {
		It("should contain the passed in version and commit information in internal data structures", func() {
			Expect(Version.Version).To(Equal(ldflagVersionOverride))
			Expect(Version.Commit).To(Equal(ldflagCommitOverride))
		})
	})

	Context("When printing the VersionContext", func() {
		It("should display the version and the commit information as a string", func() {
			Expect(strings.Contains(Version.String(), ldflagVersionOverride)).To(BeTrue())
			Expect(strings.Contains(Version.String(), ldflagCommitOverride)).To(BeTrue())
		})
	})

	// These tests confirm that we have appropriate JSON struct tags because we include
	// this in Preflight Results.
	Context("When using a VersionContext", func() {
		It("should have JSON struct tags on fields", func() {
			nf, nexists := reflect.TypeOf(&Version).Elem().FieldByName("Name") // The struct key!
			Expect(nexists).To(BeTrue())
			Expect(string(nf.Tag)).To(Equal(`json:"name"`)) // the tag

			vf, vexists := reflect.TypeOf(&Version).Elem().FieldByName("Version")
			Expect(vexists).To(BeTrue())
			Expect(string(vf.Tag)).To(Equal(`json:"version"`))

			cf, cexists := reflect.TypeOf(&Version).Elem().FieldByName("Commit")
			Expect(cexists).To(BeTrue())
			Expect(string(cf.Tag)).To(Equal(`json:"commit"`))
		})

		It("should only have three struct keys for tests to be valid", func() {
			keys := reflect.TypeOf(Version).NumField()
			Expect(keys).To(Equal(3))
		})
	})

	// These tests validate that GetLatestReleasedVersion fetches the latest available github release
	Context("When retrieving latest available release from Github", func() {
		Context("When current version is older than the latest version", func() {
			It("should return a version", func() {
				client := &MockGhVersionClientNewer{}
				release, err := Version.LatestReleasedVersion(mockCheckContainerCmd(), client)
				Expect(err).To(BeNil())
				Expect(release.TagName)
				Expect(release.HTMLURL)
			})
		})
		Context("When current version is newer than the latest version", func() {
			It("should return nil", func() {
				client := &MockGhVersionClientOlder{}
				release, err := Version.LatestReleasedVersion(mockCheckContainerCmd(), client)
				Expect(err).To(BeNil())
				Expect(release).To(BeNil())
			})
		})
		Context("When the version is not in semver format", func() {
			It("should return an error", func() {
				client := &MockGhVersionClientBadVersion{}
				release, err := Version.LatestReleasedVersion(mockCheckContainerCmd(), client)
				Expect(err).To(Not(BeNil()))
				Expect(release).To(BeNil())
			})
		})
		Context("When there is an error fetching the latest release from github", func() {
			It("should return nil", func() {
				client := &MockGhVersionClientError{}
				release, err := Version.LatestReleasedVersion(mockCheckContainerCmd(), client)
				Expect(err).To(Not(BeNil()))
				Expect(release).To(BeNil())
			})
		})
	})
})

func mockCheckContainerCmd() *cobra.Command {
	mockCheckContainerCmd := cobra.Command{}
	mockCheckContainerCmd.SetContext(context.Background())
	logger := logrusr.New(logrus.New())
	ctx := logr.NewContext(mockCheckContainerCmd.Context(), logger)
	mockCheckContainerCmd.SetContext(ctx)
	flags := mockCheckContainerCmd.Flags()
	flags.String("gh-auth-token", "", "A Github auth token can be specified to work around rate limits")
	_ = viper.BindPFlag("gh-auth-token", flags.Lookup("gh-auth-token"))
	return &mockCheckContainerCmd
}

type MockGhVersionClientNewer struct{}

type MockGhVersionClientOlder struct{}

type MockGhVersionClientError struct{}

type MockGhVersionClientBadVersion struct{}

func (mc *MockGhVersionClientNewer) GetLatestRelease(ctx context.Context, owner string, repo string) (*github.RepositoryRelease, *github.Response, error) {
	tag := "0.0.2"
	url := "test.com/release/0.0.2"

	release := github.RepositoryRelease{
		TagName: &tag,
		HTMLURL: &url,
	}
	response := github.Response{
		Rate: github.Rate{
			Limit:     60,
			Remaining: 59,
		},
	}

	return &release, &response, nil
}

func (mc *MockGhVersionClientOlder) GetLatestRelease(ctx context.Context, owner string, repo string) (*github.RepositoryRelease, *github.Response, error) {
	tag := "0.0.1"
	url := "test.com/release/0.0.1"
	release := github.RepositoryRelease{
		TagName: &tag,
		HTMLURL: &url,
	}
	response := github.Response{
		Rate: github.Rate{
			Limit:     60,
			Remaining: 59,
		},
	}

	return &release, &response, nil
}

func (mc *MockGhVersionClientBadVersion) GetLatestRelease(ctx context.Context, owner string, repo string) (*github.RepositoryRelease, *github.Response, error) {
	tag := "foobar"
	url := "test.com/release/foobar"
	release := github.RepositoryRelease{
		TagName: &tag,
		HTMLURL: &url,
	}
	response := github.Response{
		Rate: github.Rate{
			Limit:     60,
			Remaining: 59,
		},
	}

	return &release, &response, nil
}

func (mc *MockGhVersionClientError) GetLatestRelease(ctx context.Context, owner string, repo string) (*github.RepositoryRelease, *github.Response, error) {
	return nil, nil, errors.New("unspecified Error")
}
