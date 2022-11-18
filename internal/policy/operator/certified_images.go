package operator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/pyxis"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/operator-framework/operator-manifest-tools/pkg/image"
	"github.com/operator-framework/operator-manifest-tools/pkg/pullspec"
	log "github.com/sirupsen/logrus"
)

var _ certification.Check = &certifiedImagesCheck{}

// imageFinder interface is used for testing. It represents the FindImagesByDigest
// function that is part of the Pyxis client.
type imageFinder interface {
	FindImagesByDigest(context.Context, []string) ([]pyxis.CertImage, error)
}

// CertifiedImagesCheck scans the CSV and validates that all refereenced images are certified.
type certifiedImagesCheck struct {
	imageFinder        imageFinder
	nonCertifiedImages []string
}

// NewCertifiedImagesCheck returns a *certifiedImagesCheck that will scan the CSV and validate
// that all referenced images are certified.
func NewCertifiedImagesCheck(imageFinder imageFinder) *certifiedImagesCheck {
	return &certifiedImagesCheck{
		imageFinder: imageFinder,
	}
}

func (p *certifiedImagesCheck) Validate(ctx context.Context, imgRef certification.ImageReference) (bool, error) {
	imageDigests, err := p.dataToValidate(ctx, filepath.Join(imgRef.ImageFSPath, "manifests"))
	if err != nil {
		return false, err
	}

	return p.validate(ctx, imageDigests)
}

//nolint:unparam // ctx is unused. Keep for future use.
func (p *certifiedImagesCheck) dataToValidate(ctx context.Context, imagePath string) ([]string, error) {
	operatorManifests, err := pullspec.FromDirectory(imagePath, pullspec.DefaultHeuristic)
	if err != nil {
		return nil, err
	}
	imageNames, err := image.Extract(operatorManifests)
	if err != nil {
		return nil, err
	}

	imageDigests := make([]string, 0, len(imageNames))
	for _, img := range imageNames {
		digest, err := name.NewDigest(img)
		if err != nil {
			log.Warningf("Image does not appear to be pinned: %s: %v", img, err)
			p.nonCertifiedImages = append(p.nonCertifiedImages, img)
			continue
		}
		imageDigests = append(imageDigests, digest.DigestStr())
	}

	return imageDigests, nil
}

func (p *certifiedImagesCheck) validate(ctx context.Context, imageDigests []string) (bool, error) {
	pyxisImages, err := p.imageFinder.FindImagesByDigest(ctx, imageDigests)
	if err != nil {
		return false, err
	}

	foundMap := make(map[string]pyxis.CertImage, len(pyxisImages))
	for _, img := range pyxisImages {
		foundMap[img.DockerImageDigest] = img
	}

	for _, digest := range imageDigests {
		img, ok := foundMap[digest]
		if !ok {
			log.Warningf("Image not found in Pyxis, therefore it is not certified: %s", digest)
			p.nonCertifiedImages = append(p.nonCertifiedImages, digest)
			continue
		}
		if !img.Certified {
			fullImg := fmt.Sprintf("%s/%s@%s", img.Repositories[0].Registry, img.Repositories[0].Repository, img.DockerImageDigest)
			log.Warningf("Image is not certified: %s", fullImg)
			p.nonCertifiedImages = append(p.nonCertifiedImages, fullImg)
		}
	}
	return true, nil
}

func (p *certifiedImagesCheck) Name() string {
	return "BundleImageRefsAreCertified"
}

func (p *certifiedImagesCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking that all images referenced in the CSV are certified. Currently, this check is not enforced.",
		Level:            "optional",
		KnowledgeBaseURL: "https://access.redhat.com/documentation/en-us/red_hat_software_certification/8.45/html/red_hat_openshift_software_certification_policy_guide/assembly-products-managed-by-an-operator_openshift-sw-cert-policy-container-images#con-operand-requirements_openshift-sw-cert-policy-products-managed",
		CheckURL:         "https://access.redhat.com/documentation/en-us/red_hat_software_certification/8.45/html/red_hat_openshift_software_certification_policy_guide/assembly-products-managed-by-an-operator_openshift-sw-cert-policy-container-images#con-operand-requirements_openshift-sw-cert-policy-products-managed",
	}
}

func (p *certifiedImagesCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check that all images referenced in the CSV are certified.",
		Suggestion: "Ensure that any images referenced in the CSV, including the relatedImages section, have been certified.",
	}
}
