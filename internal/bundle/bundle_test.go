package bundle

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/blang/semver/v4"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/table"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
)

var _ = Describe("BundleValidateCheck", func() {
	// unmappedOCPVersion returns an OpenShift version guaranteed to have
	// no entry in ocpToKubeVersion, so specs that use it do not go stale
	// as new versions are added to the map with each release.
	unmappedOCPVersion := func() string {
		var latest semver.Version
		for v := range ocpToKubeVersion {
			parsed, err := semver.ParseTolerant(v)
			if err != nil {
				continue
			}
			if parsed.GT(latest) {
				latest = parsed
			}
		}
		return fmt.Sprintf("%d.%d", latest.Major, latest.Minor+1)
	}

	Describe("Bundle validation", func() {
		Context("the annotations file is valid", func() {
			It("should pass", func() {
				imageRef := image.ImageReference{
					ImageFSPath: "./testdata/valid_bundle",
				}
				report, err := Validate(context.Background(), imageRef.ImageFSPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(report).ToNot(BeNil())
			})
		})

		Context("the annotations file does not exist", func() {
			It("should error", func() {
				imageRef := image.ImageReference{
					ImageFSPath: "./testdata/no_annotations_file",
				}
				report, err := Validate(context.Background(), imageRef.ImageFSPath)
				Expect(err).To(HaveOccurred())
				Expect(report).To(BeNil())
			})
		})

		Context("the annotations file is malformed", func() {
			It("should error", func() {
				imageRef := image.ImageReference{
					ImageFSPath: "./testdata/malformed_annotations_file",
				}
				report, err := Validate(context.Background(), imageRef.ImageFSPath)
				Expect(err).To(HaveOccurred())
				Expect(report).To(BeNil())
			})
		})

		Context("the annotations file is valid but has no annotations", func() {
			It("should fail gracefully", func() {
				imageRef := image.ImageReference{
					ImageFSPath: "./testdata/invalid_bundle",
				}
				report, err := Validate(context.Background(), imageRef.ImageFSPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(report).ToNot(BeNil())
			})
		})

		Context("the annotations file specifies a version with no Kubernetes version mapping", func() {
			It("should pass without k8s-version-dependent checks", func() {
				// Derive a version guaranteed to be absent from
				// ocpToKubeVersion so this spec does not go stale as
				// new versions are added with each release.
				unmapped := unmappedOCPVersion()

				tmpDir, err := os.MkdirTemp("", "preflight-unmapped-bundle-*")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() { _ = os.RemoveAll(tmpDir) })
				Expect(os.CopyFS(tmpDir, os.DirFS("./testdata/valid_bundle"))).To(Succeed())

				annotations := fmt.Sprintf(`annotations:
  com.redhat.openshift.versions: "v%s"
  operators.operatorframework.io.bundle.package.v1: testPackage
  operators.operatorframework.io.bundle.channel.default.v1: testChannel
`, unmapped)
				Expect(os.WriteFile(filepath.Join(tmpDir, "metadata", "annotations.yaml"), []byte(annotations), 0o600)).To(Succeed())

				report, err := Validate(context.Background(), tmpDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(report).ToNot(BeNil())
			})
		})
	})

	Describe("While ensuring that container util is working", func() {
		// tests: extractAnnotationsBytes
		Context("with an annotations yaml data read from disk", func() {
			Context("containing no data read in from the yaml file", func() {
				data := []byte{}

				It("should return an error", func() {
					_, err := LoadAnnotations(context.TODO(), bytes.NewReader(data))
					Expect(err).To(HaveOccurred())
				})
			})

			Context("containing malformed or unexpected data", func() {
				data := []byte(`malformed`)

				It("should return an error", func() {
					_, err := LoadAnnotations(context.TODO(), bytes.NewReader(data))
					Expect(err).To(HaveOccurred())
				})
			})

			Context("a bad reader is sent to GetAnnotations", func() {
				It("should return an error", func() {
					annotations, err := LoadAnnotations(context.TODO(), errReader(0))
					Expect(err).To(HaveOccurred())
					Expect(annotations).To(BeNil())
				})
			})
		})
	})

	Describe("hasSCCApiGroup", func() {
		Context("when the rule contains security.openshift.io", func() {
			It("should return true", func() {
				rule := rbacv1.PolicyRule{
					APIGroups: []string{"", "security.openshift.io"},
				}
				Expect(hasSCCApiGroup(rule)).To(BeTrue())
			})
		})

		Context("when the rule does not contain security.openshift.io", func() {
			It("should return false", func() {
				rule := rbacv1.PolicyRule{
					APIGroups: []string{"apps", ""},
				}
				Expect(hasSCCApiGroup(rule)).To(BeFalse())
			})
		})

		Context("when the rule has no API groups", func() {
			It("should return false", func() {
				rule := rbacv1.PolicyRule{
					APIGroups: []string{},
				}
				Expect(hasSCCApiGroup(rule)).To(BeFalse())
			})
		})
	})

	Describe("hasSCCResource", func() {
		Context("when the rule contains securitycontextconstraints", func() {
			It("should return true", func() {
				rule := rbacv1.PolicyRule{
					Resources: []string{"pods", "securitycontextconstraints"},
				}
				Expect(hasSCCResource(rule)).To(BeTrue())
			})
		})

		Context("when the rule does not contain securitycontextconstraints", func() {
			It("should return false", func() {
				rule := rbacv1.PolicyRule{
					Resources: []string{"pods", "deployments"},
				}
				Expect(hasSCCResource(rule)).To(BeFalse())
			})
		})

		Context("when the rule has no resources", func() {
			It("should return false", func() {
				rule := rbacv1.PolicyRule{
					Resources: []string{},
				}
				Expect(hasSCCResource(rule)).To(BeFalse())
			})
		})
	})

	Describe("GetSecurityContextConstraints", func() {
		Context("when the bundle path is invalid", func() {
			It("should return an error", func() {
				sccs, err := GetSecurityContextConstraints(context.TODO(), "/nonexistent/path")
				Expect(err).To(HaveOccurred())
				Expect(sccs).To(BeNil())
			})
		})

		Context("when the bundle has no SCC rules", func() {
			It("should return nil", func() {
				sccs, err := GetSecurityContextConstraints(context.TODO(), "./testdata/valid_bundle")
				Expect(err).ToNot(HaveOccurred())
				Expect(sccs).To(BeNil())
			})
		})

		Context("when the bundle has SCC rules", func() {
			It("should return the SCC resource names", func() {
				sccs, err := GetSecurityContextConstraints(context.TODO(), "./testdata/scc_bundle")
				Expect(err).ToNot(HaveOccurred())
				Expect(sccs).To(ConsistOf("my-scc", "another-scc"))
			})
		})
	})

	DescribeTable("Image Registry validation",
		func(versions string, expected string, success bool) {
			version, err := targetVersion(versions)
			if success {
				Expect(err).ToNot(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
			Expect(version).To(Equal(expected))
		},

		Entry("range 4.6 to 4.8", "v4.6-v4.8", "4.8", true),
		Entry("exactly 4.8", "=v4.8", "4.8", true),
		Entry("exactly 4.9", "=v4.9", "4.9", true),
		Entry("range 4.6 to 4.9", "v4.6-v4.9", "4.9", true),
		Entry(">= 4.8, older than latest released, resolves to latest released", "v4.8", latestReleasedVersion, true),
		Entry(">= 4.9, older than latest released, resolves to latest released", "v4.9", latestReleasedVersion, true),
		Entry(">= 4.11, older than latest released, resolves to latest released", "v4.11", latestReleasedVersion, true),
		Entry(">= 4.15, older than latest released, resolves to latest released", "v4.15", latestReleasedVersion, true),
		Entry(">= 4.16, older than latest released, resolves to latest released", "v4.16", latestReleasedVersion, true),
		Entry(">= 4.17, older than latest released, resolves to latest released", "v4.17", latestReleasedVersion, true),
		Entry(">= 4.18, older than latest released, resolves to latest released", "v4.18", latestReleasedVersion, true),
		Entry(">= 4.19, older than latest released, resolves to latest released", "v4.19", latestReleasedVersion, true),
		Entry(">= 4.20, older than latest released, resolves to latest released", "v4.20", latestReleasedVersion, true),
		Entry(">= 4.21, older than latest released, resolves to latest released", "v4.21", latestReleasedVersion, true),
		Entry(">= 4.22, older than latest released, resolves to latest released", "v4.22", latestReleasedVersion, true),
		Entry(">= 4.23, older than latest released, resolves to latest released", "v4.23", latestReleasedVersion, true),
		Entry(">= 5.0, equal to latest released, resolves to latest released", "v5.0", latestReleasedVersion, true),
		Entry(">= 5.1, newer than latest released, passes through", "v5.1", "5.1", true),
		Entry("begins = with error", "=foo", "", false),
		Entry("bare version with error", "vfoo", "", false),
		Entry("range with error", "v4.6-vfoo", "", false),
		Entry("open-ended range is error", "v4.11-", "", false),
	)
})
