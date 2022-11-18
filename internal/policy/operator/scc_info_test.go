package operator

import (
	"context"
	"os"
	"path/filepath"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("securityContextConstraintsInCSV", func() {
	const (
		manifestsDir                  = "manifests"
		clusterServiceVersionFilename = "myoperator.clusterserviceversion.yaml"
		customSCC                     = "\n      clusterPermissions:\n      - rules:\n        - apiGroups:\n          - security.openshift.io\n          resources:\n          - securitycontextconstraints\n          resourceNames:\n          - custom\n          verbs:\n          - use\n        serviceAccountName: my-operand\n"
		multipleSCC                   = "\n      clusterPermissions:\n      - rules:\n        - apiGroups:\n          - security.openshift.io\n          resources:\n          - securitycontextconstraints\n          resourceNames:\n          - anyuid\n          - nonroot\n          verbs:\n          - use\n        serviceAccountName: my-operand\n"
		noSCC                         = "\n"
	)
	var (
		securityContextConstraintsCheck securityContextConstraintsInCSV
		imageRef                        certification.ImageReference
		csvContents                     = buildCsvContent(noSCC)
	)
	AssertMetaData(&securityContextConstraintsCheck)
	BeforeEach(func() {
		securityContextConstraintsCheck = *NewSecurityContextConstraintsCheck()
		tmpDir, err := os.MkdirTemp("", "scc-check-bundle-*")
		Expect(err).ToNot(HaveOccurred())
		imageRef.ImageFSPath = tmpDir
		DeferCleanup(os.RemoveAll, tmpDir)
		err = os.Mkdir(filepath.Join(tmpDir, manifestsDir), 0o755)
		Expect(err).ToNot(HaveOccurred())
		err = os.WriteFile(filepath.Join(tmpDir, manifestsDir, clusterServiceVersionFilename), []byte(csvContents), 0o644)
		Expect(err).ToNot(HaveOccurred())
	})
	When("given a csv with a custom scc", func() {
		It("should succeed", func() {
			csvContents := buildCsvContent(customSCC)
			Expect(os.WriteFile(filepath.Join(imageRef.ImageFSPath, manifestsDir, clusterServiceVersionFilename), []byte(csvContents), 0o644)).To(Succeed())
			result, err := securityContextConstraintsCheck.Validate(context.TODO(), imageRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
			Expect(securityContextConstraintsCheck.Help().Message).To(ContainSubstring("custom"))
		})
	})
	When("given a csv with a multiples sccs", func() {
		It("should fail", func() {
			csvContents := buildCsvContent(multipleSCC)
			Expect(os.WriteFile(filepath.Join(imageRef.ImageFSPath, manifestsDir, clusterServiceVersionFilename), []byte(csvContents), 0o644)).To(Succeed())
			result, err := securityContextConstraintsCheck.Validate(context.TODO(), imageRef)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeFalse())
		})
	})
	When("there is no CSV", func() {
		It("should fail", func() {
			Expect(os.RemoveAll(imageRef.ImageFSPath)).To(Succeed())
			result, err := securityContextConstraintsCheck.Validate(context.TODO(), imageRef)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeFalse())
		})
	})
	When("the CSV is malformed", func() {
		It("should fail", func() {
			csvContents := `kind: ClusterServiceVersion
apiVersion: operators.coreos.com/v1alpha1
spec: malformed
`
			Expect(os.WriteFile(filepath.Join(imageRef.ImageFSPath, manifestsDir, clusterServiceVersionFilename), []byte(csvContents), 0o644)).To(Succeed())
			result, err := securityContextConstraintsCheck.Validate(context.TODO(), imageRef)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeFalse())
		})
	})
	When("no scc is specfied in csv", func() {
		It("should still pass", func() {
			csvContents := buildCsvContent(noSCC)
			Expect(os.WriteFile(filepath.Join(imageRef.ImageFSPath, manifestsDir, clusterServiceVersionFilename), []byte(csvContents), 0o644)).To(Succeed())
			result, err := securityContextConstraintsCheck.Validate(context.TODO(), imageRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})
	})
})

func buildCsvContent(sccBlock string) string {
	return `kind: ClusterServiceVersion
apiVersion: operators.coreos.com/v1alpha1
spec:
  install:
    spec:
      deployments:
      - spec:
          template:
            spec:
              containers:
              - image: registry.example.io/foo/bar@sha256:f000432f07cd187469f0310e3ed9dcf9a5db2be14b8bab9c5293dd1ee8518176
                name: the-operator` + sccBlock +
		`relatedImages:
  - name: the-operator
    image: registry.example.io/foo/bar@sha256:f000432f07cd187469f0310e3ed9dcf9a5db2be14b8bab9c5293dd1ee8518176
  - name: the-proxy
    image: registry.example.io/foo/proxy@sha256:5e33f9d095952866b9743cc8268fb740cce6d93439f00ce333a2de1e5974837e`
}
