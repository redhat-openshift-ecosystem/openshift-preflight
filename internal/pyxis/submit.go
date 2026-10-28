package pyxis

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/name"
)

var defaultRegistryAlias = "docker.io"

// SubmitResults takes certInput and sends requests to Pyxis to create or update entries
// based on certInput. certInput is not modified.
//
// The image is created without its certification status, and that status is applied
// via updateImage only after the RPM manifest, artifacts, and test results have been
// submitted. Pyxis' auto-publish reacts to the certified flag, so deferring it prevents
// a later failure in this flow from publishing an image with incomplete data. If the
// image is already certified (e.g. a previous submission succeeded), the update is
// skipped since Pyxis rejects updates to published images.
func (p *pyxisClient) SubmitResults(ctx context.Context, certInput *CertificationInput) (*CertificationResults, error) {
	var err error

	// Work on copies of the input so callers can safely reuse it. The flow below
	// clears and re-applies the image's certified flag and normalizes registries.
	certImageCopy := *certInput.CertImage
	certImageCopy.Repositories = slices.Clone(certInput.CertImage.Repositories)
	certImage := &certImageCopy

	// You must have an existing repository.
	if len(certImage.Repositories) == 0 {
		return nil, fmt.Errorf("certImage has not been properly populated")
	}

	// Create the test results, so we can fail fast if version check throws error.
	testResultsCopy := *certInput.TestResults
	testResults := &testResultsCopy

	testResults, err = p.createTestResults(ctx, testResults)
	if err != nil {
		return nil, fmt.Errorf("could not create test results: %v", err)
	}

	certProjectCopy := *certInput.CertProject
	certProject := &certProjectCopy

	// Submission effectively starts the certification process, so switch
	// the status to reflect this if needed. This only needs to be done for net new projects.
	// Existing projects that are in "In Progress" can stay "In Progress" until they moved to "Published" which is triggered
	// once an image in a project is moved to "Published" status. The status on the project would stay in "Published" status,
	// unless the partner decides to un-publish all of their images. At that point backed systems/processes would move
	// the project back to "In Process" and there would still be nothing that preflight need to update on the project.
	if certProject.CertificationStatus == "Started" {
		certProject.CertificationStatus = "In Progress"
	}

	// Always set the project's metadata to match the image that we're certifying. These values will always be sent
	// to pyxis which has the validation rules on if the values can be updated, and will throw an exception if they
	// are not allowed to be updated, ie if the images/projects are already published.
	// Also normalizing index.docker.io to docker.io for the certProject
	certProject.Container.Registry = normalizeDockerRegistry(certImage.Repositories[0].Registry)
	certProject.Container.Repository = certImage.Repositories[0].Repository

	// always update the project no matter the status to ensure the dockerconfig preflight used to pull the image
	// is the dockerfile that resides on the project and other backend processes ie clair use the same file
	// Note: users no longer have the ability to update their project's dockerconfig in connect
	certProject, err = p.updateProject(ctx, certProject)
	if err != nil {
		return nil, fmt.Errorf("could not update project: %v", err)
	}

	// store the original digest so that we can pull the image later
	// in the event that it exists. createImage will wipe it otherwise.
	originalImageDigest := certImage.DockerImageDigest

	// Pyxis' auto-publish reacts to the certified flag, so keep it out of the
	// initial image creation and apply it only after all supporting data is
	// submitted (see the updateImage call at the end of this function).
	certified := certImage.Certified
	certImage.Certified = false

	// normalizing index.docker.io to docker.io for the certImage
	certImage.Repositories[0].Registry = normalizeDockerRegistry(certImage.Repositories[0].Registry)

	// Create the image, or get it if it already exists.
	certImage, err = p.createImage(ctx, certImage)
	if err != nil {
		if !errors.Is(err, ErrPyxis409StatusCode) {
			return nil, fmt.Errorf("could not create image: %v", err)
		}
		certImage, err = p.getImage(ctx, originalImageDigest)
		if err != nil {
			return nil, fmt.Errorf("could not get image: %v", err)
		}
	}

	if !certProject.ScratchProject() {
		// Create the RPM manifest, or get it if it already exists.
		// Work on a copy so linking the image ID doesn't mutate the caller's input.
		rpmManifest := *certInput.RpmManifest
		rpmManifest.ImageID = certImage.ID
		_, err = p.createRPMManifest(ctx, &rpmManifest)
		if err != nil {
			if !errors.Is(err, ErrPyxis409StatusCode) {
				return nil, fmt.Errorf("could not create rpm manifest: %v", err)
			}
			_, err = p.getRPMManifest(ctx, rpmManifest.ImageID)
			if err != nil {
				return nil, fmt.Errorf("could not get rpm manifest: %v", err)
			}
		}
	}

	// Create the artifacts in Pyxis.
	artifacts := certInput.Artifacts
	for _, artifact := range artifacts {
		artifact.ImageID = certImage.ID
		if _, err := p.createArtifact(ctx, &artifact); err != nil {
			return nil, fmt.Errorf("could not create artifact: %s: %v", artifact.Filename, err)
		}
	}

	// Update the test results with the certification image id to link the results to the image.
	testResults.ImageID = certImage.ID

	testResults, err = p.updateTestResults(ctx, testResults)
	if err != nil {
		return nil, fmt.Errorf("could not update test results: %v", err)
	}

	// All supporting data is now in place, so apply the certification status.
	// This is the write that can trigger auto-publish in Pyxis, hence last.
	if certified {
		certImage, err = p.certifyImage(ctx, certImage)
		if err != nil {
			return nil, fmt.Errorf("could not update image certification status: %v. "+
				"All test results and image data were submitted successfully; re-running "+
				"preflight check container with --submit will complete the submission", err)
		}
	}

	// Return the results with up-to-date information.
	return &CertificationResults{
		CertProject: certProject,
		CertImage:   certImage,
		TestResults: testResults,
	}, nil
}

// certifyImage applies the certified flag to an image, which is the write that can
// trigger auto-publish in Pyxis. An already-certified image is returned unchanged,
// since the flag is already applied and Pyxis rejects updates to published images.
func (p *pyxisClient) certifyImage(ctx context.Context, certImage *CertImage) (*CertImage, error) {
	if certImage.Certified {
		logr.FromContextOrDiscard(ctx).Info("Image is already certified; skipping certification update", "image_id", certImage.ID)
		return certImage, nil
	}
	certImage.Certified = true

	return p.updateImage(ctx, certImage)
}

// normalizeDockerRegistry sets registry to the value we get from certImage from crane and then normalizes
// index.docker.io to docker.io so project/image info shows properly in the Red Hat Catalog and other backend systems (Clair)
func normalizeDockerRegistry(registry string) string {
	if registry == name.DefaultRegistry {
		registry = defaultRegistryAlias
	}

	return registry
}
