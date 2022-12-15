package container

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
)

var _ = Describe("UniqueTag", func() {
	hasUniqueTagCheck := *NewHasUniqueTagCheck("")
	var src, dst, host string

	BeforeEach(func() {
		// Set up a fake registry.
		registryLogger := log.New(io.Discard, "", log.Ldate)
		s := httptest.NewServer(registry.New(registry.Logger(registryLogger)))
		DeferCleanup(s.Close)
		u, err := url.Parse(s.URL)
		Expect(err).ToNot(HaveOccurred())
		src = fmt.Sprintf("%s/test/preflight", u.Host)
		dst = fmt.Sprintf("%s/test/tags", u.Host)
		host = u.Host

		img, err := random.Image(1024, 5)
		Expect(err).ToNot(HaveOccurred())

		err = crane.Push(img, src)
		Expect(err).ToNot(HaveOccurred())

		err = crane.Copy(src, dst)
		Expect(err).ToNot(HaveOccurred())

		err = crane.Tag(dst, "unique-tag")
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("Checking for unique tags", func() {
		Context("When it has tags other than latest", func() {
			It("should pass Validate", func() {
				ok, err := hasUniqueTagCheck.Validate(context.TODO(), image.ImageReference{ImageRegistry: host, ImageRepository: "test/tags", ImageTagOrSha: "sha256:12345"})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When it has tags other than latest and registry throws an error", func() {
			BeforeEach(func() {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
				DeferCleanup(s.Close)

				u, err := url.Parse(s.URL)
				Expect(err).ToNot(HaveOccurred())
				dst = fmt.Sprintf("%s/test/tags", u.Host)
				host = u.Host
			})
			It("should throw an error", func() {
				ok, err := hasUniqueTagCheck.Validate(context.TODO(), image.ImageReference{ImageRegistry: host, ImageRepository: "test/tags", ImageTagOrSha: "sha256:12345"})
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})

		Context("When it has only latest tag", func() {
			It("should not pass Validate", func() {
				ok, err := hasUniqueTagCheck.Validate(context.TODO(), image.ImageReference{ImageRegistry: host, ImageRepository: "test/preflight", ImageTagOrSha: "latest"})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		Context("When it has only latest tag and the registry throws an error", func() {
			BeforeEach(func() {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
				DeferCleanup(s.Close)

				u, err := url.Parse(s.URL)
				Expect(err).ToNot(HaveOccurred())
				dst = fmt.Sprintf("%s/test/tags", u.Host)
				host = u.Host
			})
			It("should throw an error", func() {
				ok, err := hasUniqueTagCheck.Validate(context.TODO(), image.ImageReference{ImageRegistry: host, ImageRepository: "test/preflight", ImageTagOrSha: "latest"})
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		Context("When registry returns an empty tag list", func() {
			BeforeEach(func() {
				s := httptest.NewServer(http.HandlerFunc(mockRegistry))
				DeferCleanup(s.Close)

				u, err := url.Parse(s.URL)
				Expect(err).ToNot(HaveOccurred())
				host = u.Host
			})
			It("should pass Validate", func() {
				ok, err := hasUniqueTagCheck.Validate(context.TODO(), image.ImageReference{ImageRegistry: host, ImageRepository: "test/notags", ImageTagOrSha: "v0.0.1"})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
			It("should fail Validate", func() {
				ok, err := hasUniqueTagCheck.Validate(context.TODO(), image.ImageReference{ImageRegistry: host, ImageRepository: "test/notags", ImageTagOrSha: "sha256:12345"})
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})

	AssertMetaData(&hasUniqueTagCheck)
})

func emptyImageTags() []string {
	return []string{}
}

type tagsList struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

// mockRegistry() is customized due to the way some partners don't expose
// the `/tags/list` API endpoint in their private registry implementations.
func mockRegistry(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	repo := "test/notags"
	tagURLPath := "/v2/" + repo + "/tags/list"
	if req.URL.Path != tagURLPath && req.URL.Path != tagURLPath+"/" && req.Method != "GET" {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	resp.Header().Set("Content-Type", "application/json")

	tagsResp := tagsList{
		Name: repo,
		Tags: emptyImageTags(),
	}

	jbod, _ := json.Marshal(tagsResp)
	resp.Header().Set("Content-Length", fmt.Sprint(len(jbod)))
	resp.WriteHeader(http.StatusOK)
	_, _ = io.Copy(resp, bytes.NewReader(jbod))
}
