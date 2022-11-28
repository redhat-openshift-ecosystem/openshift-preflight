package pyxis

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pyxis Builder tests", func() {
	var tmpdir string
	var err error

	BeforeEach(func() {
		// create tmpdir to receive extracted fs
		tmpdir, err = os.MkdirTemp(os.TempDir(), "builder-test-*")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := os.RemoveAll(tmpdir)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("When preparing a new input builder", func() {
		Context("with a nil CertProject value", func() {
			It("should return an error", func() {
				_, err := NewCertificationInput(context.Background(), nil)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with a non-nil CertProject value as input", func() {
			var p *CertProject
			var certImage *bytes.Buffer
			var results *bytes.Buffer
			var rpmManifest *bytes.Buffer

			BeforeEach(func() {
				p = &CertProject{
					ID:                  "myID",
					CertificationStatus: "Pending",
					Container: Container{
						DockerConfigJSON: "",
						Type:             "",
						ISVPID:           "",
						Registry:         "",
						Repository:       "",
						OsContentType:    "",
						Privileged:       false,
					},
					Name:          "My Project",
					ProjectStatus: "Some Status",
					Type:          "Container",
				}
				certImage = bytes.NewBufferString(`{"id":"foo","certified":false,"deleted":false}`)
				results = bytes.NewBufferString(`{}`)
				rpmManifest = bytes.NewBufferString(`{}`)
			})

			It("should not return an error", func() {
				_, err := NewCertificationInput(context.Background(), p,
					WithCertImage(certImage),
					WithPreflightResults(results),
					WithRPMManifest(rpmManifest),
				)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return a certification input builder with the project embedded", func() {
				input, err := NewCertificationInput(context.Background(), p,
					WithCertImage(certImage),
					WithPreflightResults(results),
					WithRPMManifest(rpmManifest),
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(input.CertProject.ID).To(Equal(p.ID))
			})

			It("should not bind with an invalid io.Reader", func() {
				_, err := NewCertificationInput(context.Background(), p,
					WithCertImage(errReader(0)),
					WithPreflightResults(results),
					WithRPMManifest(rpmManifest),
				)
				Expect(err).To(HaveOccurred())
			})

			It("should not bind an invalid cert image", func() {
				_, err := NewCertificationInput(context.Background(), p,
					WithCertImage(bytes.NewBufferString("\tHello world!\n")),
					WithPreflightResults(results),
					WithRPMManifest(rpmManifest),
				)
				Expect(err).To(HaveOccurred())
			})

			It("should not bind an empty cert image", func() {
				_, err := NewCertificationInput(context.Background(), p,
					WithCertImage(bytes.NewBufferString("")),
					WithPreflightResults(results),
					WithRPMManifest(rpmManifest),
				)
				Expect(err).To(HaveOccurred())
			})

			It("should fail with no test results", func() {
				_, err := NewCertificationInput(context.Background(), p,
					WithCertImage(bytes.NewBufferString("")),
					WithRPMManifest(rpmManifest),
				)
				Expect(err).To(HaveOccurred())
			})

			It("should allow binding a valid preflight result read from a file", func() {
				results := TestResults{
					ID:          "foo",
					CertProject: "",
					OrgID:       0,
					Version:     "",
					ImageID:     "",
					UserResponse: formatters.UserResponse{
						Image:             "bar",
						Passed:            false,
						CertificationHash: "",
						LibraryInfo:       version.VersionContext{},
					},
				}
				bts, err := json.Marshal(results)
				Expect(err).ToNot(HaveOccurred())

				input, err := NewCertificationInput(context.Background(), p,
					WithCertImage(certImage),
					WithPreflightResults(bytes.NewBuffer(bts)),
					WithRPMManifest(rpmManifest),
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(input.TestResults.ID).To(Equal(results.ID))
				Expect(input.TestResults.UserResponse.Image).To(Equal(results.UserResponse.Image))
			})

			It("should not bind an invalid preflight results from file", func() {
				_, err := NewCertificationInput(context.Background(), p,
					WithCertImage(certImage),
					WithPreflightResults(bytes.NewBufferString("\tHello world!\n")),
					WithRPMManifest(rpmManifest),
				)
				Expect(err).To(HaveOccurred())
			})

			It("should fail to finalize with no rpm manifest", func() {
				_, err := NewCertificationInput(context.Background(), p,
					WithCertImage(certImage),
					WithPreflightResults(results),
				)
				Expect(err).To(HaveOccurred())
			})

			It("should allow binding a valid rpmmanifest read from a file", func() {
				rpmManifest := RPMManifest{
					ID:      "foo",
					ImageID: "bar",
					RPMS: []RPM{{
						Architecture: "",
						Gpg:          "",
						Name:         "baz",
						Nvra:         "",
						Release:      "",
						SrpmName:     "",
						SrpmNevra:    "",
						Summary:      "",
						Version:      "",
					}},
				}
				bts, err := json.Marshal(rpmManifest)
				Expect(err).ToNot(HaveOccurred())

				input, err := NewCertificationInput(context.Background(), p,
					WithCertImage(certImage),
					WithPreflightResults(results),
					WithRPMManifest(bytes.NewBuffer(bts)),
				)

				Expect(err).ToNot(HaveOccurred())
				Expect(input.RpmManifest.ID).To(Equal(rpmManifest.ID))
				Expect(input.RpmManifest.ImageID).To(Equal(rpmManifest.ImageID))
				Expect(input.RpmManifest.RPMS[0].Name).To(Equal(rpmManifest.RPMS[0].Name))
			})

			It("should not bind an invalid rpm manifest from file", func() {
				_, err := NewCertificationInput(context.Background(), p,
					WithCertImage(certImage),
					WithPreflightResults(results),
					WithRPMManifest(bytes.NewBufferString("\tHello world!\n")),
				)
				Expect(err).To(HaveOccurred())
			})

			It("should finalize successfully with no artifacts", func() {
				input, err := NewCertificationInput(context.Background(), p,
					WithCertImage(certImage),
					WithPreflightResults(results),
					WithRPMManifest(rpmManifest),
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(input.Artifacts)).To(BeZero())
			})

			It("should bind an arbitrary test file as an artifact", func() {
				f := "artifact.log"
				data := "\tartifact contents\n"
				input, err := NewCertificationInput(context.Background(), p,
					WithCertImage(certImage),
					WithPreflightResults(results),
					WithRPMManifest(rpmManifest),
					WithArtifact(bytes.NewBufferString(data), f),
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(input.Artifacts)).To(Equal(1))
				Expect(input.Artifacts[0].Filename).To(Equal(path.Base(f)))
				Expect(int64(len(data))).To(Equal(input.Artifacts[0].FileSize))
			})

			It("should not bind an artifact with a bad Reader", func() {
				_, err := NewCertificationInput(context.Background(), p,
					WithCertImage(certImage),
					WithPreflightResults(results),
					WithRPMManifest(rpmManifest),
					WithArtifact(errReader(0), "bad-reader.txt"),
				)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
