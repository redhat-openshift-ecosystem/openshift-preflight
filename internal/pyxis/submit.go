package pyxis

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
)

var defaultRegistryAlias = "docker.io"

// SubmitResults takes certInput and sends requests to Pyxis to create or update entries
// based on certInput.
func (p *pyxisClient) SubmitResults(ctx context.Context, certInput *CertificationInput) (*CertificationResults, error) {
	var err error

	certProject := certInput.CertProject
	certImage := certInput.CertImage

	// Submission effectively starts the certification process, so switch
	// the status to reflect this if needed. This only needs to be done for net new projects.
	// Existing projects that are in "In Progress" can stay "In Progress" until they moved to "Published" which is triggered
	// once an image in a project is moved to "Published" status. The status on the project would stay in "Published" status,
	// unless the partner decides to un-publish all of their images. At that point backed systems/processes would move
	// the project back to "In Process" and there would still be nothing that preflight need to update on the project.
	if certProject.CertificationStatus == "Started" {
		certProject.CertificationStatus = "In Progress"
	}

	// You must have an existing repository.
	if len(certImage.Repositories) == 0 {
		return nil, fmt.Errorf("certImage has not been properly populated")
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

	// store the certification status for this execution, in case a previous execution failed and we need to patch the image
	certified := certInput.CertImage.Certified

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

		// checking to see if the original value is certified and the previous value is not certified,
		// this would indicate that a partner is running preflight again, and during the first run there was a timeout/error
		// in a check that interacts with pyxis and we need to correct the certified value for the image
		if certified && !certImage.Certified {
			// change the certified value to `true`
			certImage.Certified = certified

			certImage, err = p.updateImage(ctx, certImage)
			if err != nil {
				return nil, fmt.Errorf("could not update image: %v", err)
			}
		}
	}

	// Create the RPM manifest, or get it if it already exists.
	rpmManifest := certInput.RpmManifest
	rpmManifest.ImageID = certImage.ID
	_, err = p.createRPMManifest(ctx, rpmManifest)
	if err != nil {
		if !errors.Is(err, ErrPyxis409StatusCode) {
			return nil, fmt.Errorf("could not create rpm manifest: %v", err)
		}
		_, err = p.getRPMManifest(ctx, rpmManifest.ImageID)
		if err != nil {
			return nil, fmt.Errorf("could not get rpm manifest: %v", err)
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

	// Create the test results.
	testResults := certInput.TestResults
	testResults.ImageID = certImage.ID
	testResults, err = p.createTestResults(ctx, testResults)
	if err != nil {
		return nil, fmt.Errorf("could not create test results: %v", err)
	}

	// Return the results with up-to-date information.
	return &CertificationResults{
		CertProject: certProject,
		CertImage:   certImage,
		TestResults: testResults,
	}, nil
}

// normalizeDockerRegistry sets registry to the value we get from certImage from crane and then normalizes
// index.docker.io to docker.io so project/image info shows properly in the Red Hat Catalog and other backend systems (Clair)
func normalizeDockerRegistry(registry string) string {
	if registry == name.DefaultRegistry {
		registry = defaultRegistryAlias
	}

	return registry
}
