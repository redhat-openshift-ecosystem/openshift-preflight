package pyxis

import (
	"bytes"
	"encoding/json"
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"
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
				_, err := NewCertificationInput(nil)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with a non-nil CertProject value as input", func() {
			p := &CertProject{
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

			builder, err := NewCertificationInput(p)

			It("should not return an error", func() {
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return a certification input builder with the project embedded", func() {
				Expect(builder.CertificationInput.CertProject.ID).To(Equal(p.ID))
			})

			It("should fail to finalize with no cert image", func() {
				_, err := builder.Finalize()
				Expect(err).To(HaveOccurred())
			})

			It("should allow binding a valid cert image", func() {
				certimg := CertImage{
					ID:                     "foo",
					Certified:              false,
					Deleted:                false,
					DockerImageDigest:      "",
					DockerImageID:          "",
					ImageID:                "",
					ISVPID:                 "",
					ParsedData:             &ParsedData{},
					Architecture:           "",
					RawConfig:              "",
					Repositories:           []Repository{},
					SumLayerSizeBytes:      0,
					UncompressedTopLayerId: "",
				}
				bts, err := json.Marshal(certimg)
				Expect(err).ToNot(HaveOccurred())

				origBuilder := *builder
				builder = builder.WithCertImage(bytes.NewBuffer(bts))
				Expect(*builder).To(Not(Equal(origBuilder)))

				Expect(builder.CertImage.ID).To(Equal(certimg.ID))
			})

			It("should not bind with an invalid io.Reader", func() {
				origBuilder := *builder
				builder = builder.WithCertImage(errReader(0))
				Expect(*builder).To(Equal(origBuilder))
			})

			It("should not bind an invalid cert image", func() {
				origBuilder := *builder
				builder = builder.WithCertImage(bytes.NewBufferString("\tHello world!\n"))
				Expect(*builder).To(Equal(origBuilder))
			})

			It("should not bind an empty cert image", func() {
				origBuilder := *builder
				builder = builder.WithCertImage(bytes.NewBufferString(""))
				Expect(*builder).To(Equal(origBuilder))
			})

			It("should fail to finalize with no test results", func() {
				_, err := builder.Finalize()
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

				origBuilder := *builder
				builder = builder.WithPreflightResults(bytes.NewBuffer(bts))
				Expect(*builder).To(Not(Equal(origBuilder)))

				Expect(builder.TestResults.ID).To(Equal(results.ID))
				Expect(builder.TestResults.UserResponse.Image).To(Equal(results.UserResponse.Image))
			})

			It("should not bind an invalid preflight results from file", func() {
				origBuilder := *builder
				builder = builder.WithPreflightResults(bytes.NewBufferString("\tHello world!\n"))
				Expect(*builder).To(Equal(origBuilder))
			})

			It("should fail to finalize with no rpm manifest", func() {
				_, err := builder.Finalize()
				Expect(err).To(HaveOccurred())
			})

			It("should allow binding a valid rpmmanifest read from a file", func() {
				rpmmanifest := RPMManifest{
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
				bts, err := json.Marshal(rpmmanifest)
				Expect(err).ToNot(HaveOccurred())

				origBuilder := *builder
				builder = builder.WithRPMManifest(bytes.NewBuffer(bts))
				Expect(*builder).To(Not(Equal(origBuilder)))

				Expect(builder.RpmManifest.ID).To(Equal(rpmmanifest.ID))
				Expect(builder.RpmManifest.ImageID).To(Equal(rpmmanifest.ImageID))
				Expect(builder.RpmManifest.RPMS[0].Name).To(Equal(rpmmanifest.RPMS[0].Name))
			})

			It("should not bind an invalid rpm manifest from file", func() {
				origBuilder := *builder
				builder = builder.WithRPMManifest(bytes.NewBufferString("\tHello world!\n"))
				Expect(*builder).To(Equal(origBuilder))
			})

			It("should finalize successfully with no artifacts", func() {
				fbuilder, err := builder.Finalize()
				Expect(err).ToNot(HaveOccurred())

				Expect(len(fbuilder.Artifacts)).To(BeZero())
			})

			It("should bind an arbitrary test file as an artifact", func() {
				f := "artifact.log"
				data := "\tartifact contents\n"
				builder.WithArtifact(bytes.NewBufferString(data), f)

				Expect(len(builder.Artifacts)).To(Equal(1))
				Expect(builder.Artifacts[0].Filename).To(Equal(path.Base(f)))
				Expect(int64(len(data))).To(Equal(builder.Artifacts[0].FileSize))
			})

			It("should not bind an artifact with a bad Reader", func() {
				origBuilder := *builder
				builder = builder.WithArtifact(errReader(0), "bad-reader.txt")
				Expect(*builder).To(Equal(origBuilder))
			})
		})
	})
})
