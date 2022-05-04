package pyxis

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"

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

	Context("When reading a file with ReadFile", func() {
		It("should be the same size as file.Stat().Size()", func() {
			f := filepath.Join(tmpdir, "test.txt")
			os.WriteFile(f, []byte("\tHello world!\n"), 0o0755)

			file, err := os.Open(f)
			Expect(err).ToNot(HaveOccurred())

			info, err := file.Stat()
			Expect(err).ToNot(HaveOccurred())

			fileBytes, err := os.ReadFile(f)
			Expect(err).ToNot(HaveOccurred())
			Expect(int64(len(fileBytes))).To(Equal(info.Size()))
		})
	})

	Context("When reading and storing files from disk", func() {
		f := filepath.Join(tmpdir, "does-not-exist")
		It("should fail when the file does not exist", func() {
			err := readAndUnmarshal(f, nil)
			Expect(err).To(HaveOccurred())
		})
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
				Expect(builder.certificationInput.CertProject.ID).To(Equal(p.ID))
			})

			It("should fail to finalize with no cert image", func() {
				_, err := builder.Finalize()
				Expect(err).To(HaveOccurred())
			})

			It("should fail to bind a certimage from a file that does not exist", func() {
				f := filepath.Join(tmpdir, "does-not-exist")
				builder.WithCertImageFromFile(f)
				Expect(builder.CertImage).To(BeNil())
			})

			It("should allow binding a valid cert image read from a file", func() {
				f := filepath.Join(tmpdir, "certimage")
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
				os.WriteFile(f, bts, 0o0755)

				err = builder.storeCertImage(f)
				Expect(err).ToNot(HaveOccurred())

				Expect(builder.CertImage.ID).To(Equal(certimg.ID))
			})

			It("should not bind an invalid cert image from file", func() {
				f := filepath.Join(tmpdir, "certimage.invalid ")
				err := os.WriteFile(f, []byte("\tHello world!\n"), 0o0755)
				Expect(err).ToNot(HaveOccurred())

				err = builder.storeCertImage(f)
				Expect(err).To(HaveOccurred())
			})

			It("should fail to finalize with no test results", func() {
				_, err := builder.Finalize()
				Expect(err).To(HaveOccurred())
			})

			It("should fail to bind a preflight result from a file that does not exist", func() {
				f := filepath.Join(tmpdir, "does-not-exist")
				builder.WithPreflightResultsFromFile(f)
				Expect(builder.TestResults).To(BeNil())
			})

			It("should allow binding a valid preflight result read from a file", func() {
				f := filepath.Join(tmpdir, "results")
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
				os.WriteFile(f, bts, 0o0755)

				err = builder.storePreflightResults(f)
				Expect(err).ToNot(HaveOccurred())

				Expect(builder.TestResults.ID).To(Equal(results.ID))
				Expect(builder.TestResults.UserResponse.Image).To(Equal(results.UserResponse.Image))
			})

			It("should not bind an invalid preflight results from file", func() {
				f := filepath.Join(tmpdir, "results.invalid ")
				err := os.WriteFile(f, []byte("\tHello world!\n"), 0o0755)
				Expect(err).ToNot(HaveOccurred())

				err = builder.storePreflightResults(f)
				Expect(err).To(HaveOccurred())
			})

			It("should fail to finalize with no rpm manifest", func() {
				_, err := builder.Finalize()
				Expect(err).To(HaveOccurred())
			})

			It("should fail to bind a rpm manifest from a file that does not exist", func() {
				f := filepath.Join(tmpdir, "does-not-exist")
				builder.WithRPMManifestFromFile(f)
				Expect(builder.RpmManifest).To(BeNil())
			})

			It("should allow binding a valid rpmmanifest read from a file", func() {
				f := filepath.Join(tmpdir, "rpmmanfest")
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
				os.WriteFile(f, bts, 0o0755)

				err = builder.storeRPMManifest(f)
				Expect(err).ToNot(HaveOccurred())

				Expect(builder.RpmManifest.ID).To(Equal(rpmmanifest.ID))
				Expect(builder.RpmManifest.ImageID).To(Equal(rpmmanifest.ImageID))
				Expect(builder.RpmManifest.RPMS[0].Name).To(Equal(rpmmanifest.RPMS[0].Name))
			})

			It("should not bind an invalid rpm manifest from file", func() {
				f := filepath.Join(tmpdir, "rpmmanifest.invalid")
				err := os.WriteFile(f, []byte("\tHello world!\n"), 0o0755)
				Expect(err).ToNot(HaveOccurred())

				err = builder.storeRPMManifest(f)
				Expect(err).To(HaveOccurred())
			})

			It("should finalize successfully with no artifacts", func() {
				fbuilder, err := builder.Finalize()
				Expect(err).ToNot(HaveOccurred())

				Expect(len(fbuilder.Artifacts)).To(BeZero())
			})

			It("should fail to bind an artifact from a file that does not exist", func() {
				f := filepath.Join(tmpdir, "does-not-exist")
				builder.WithArtifactFromFile(f)
				Expect(len(builder.Artifacts)).To(BeZero())
			})

			It("should bind an arbitrary test file as an artifact", func() {
				f := filepath.Join(tmpdir, "artifact")
				data := []byte("\tartifact contents\n")
				err := os.WriteFile(f, data, 0o0755)
				Expect(err).ToNot(HaveOccurred())

				builder.WithArtifactFromFile(f)

				Expect(len(builder.Artifacts)).To(Equal(1))
				Expect(builder.Artifacts[0].Filename).To(Equal(path.Base(f)))
				Expect(int64(len(data))).To(Equal(builder.Artifacts[0].FileSize))
			})
		})
	})
})
