package pyxis

import (
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	log "github.com/sirupsen/logrus"
)

var defaultRegistryAlias = "docker.io"

// SubmitResults takes certInput and sends requests to Pyxis to create or update entries
// based on certInput.
func (p *pyxisClient) SubmitResults(ctx context.Context, certInput *certificationInput) (*CertificationResults, error) {
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
		return nil, errors.ErrInvalidCertImage
	}

	// Set this project's metadata to match the image that we're certifying.
	if certProject.Container.Registry == "" {
		// Setting registry to the value we get from certImage from crane and then normalizing
		// index.docker.io to docker.io so project info shows properly in the Red Hat Catalog
		registry := certImage.Repositories[0].Registry
		if registry == name.DefaultRegistry {
			registry = defaultRegistryAlias
		}

		certProject.Container.Registry = registry
	}

	if certProject.Container.Repository == "" {
		certProject.Container.Repository = certImage.Repositories[0].Repository
	}

	// always update the project no matter the status to ensure the dockerconfig preflight used to pull the image
	// is the dockerfile that resides on the project and other backend processes ie clair use the same file
	// Note: users no longer have the ability to update their project's dockerconfig in connect
	certProject, err = p.updateProject(ctx, certProject)
	if err != nil {
		log.Error(err, "could not update project")
		return nil, err
	}

	// store the original digest so that we can pull the image later
	// in the event that it exists. createImage will wipe it otherwise.
	originalImageDigest := certImage.DockerImageDigest

	// Create the image, or get it if it already exists.
	certImage, err = p.createImage(ctx, certImage)
	if err != nil {
		if err != errors.ErrPyxis409StatusCode {
			log.Error(fmt.Errorf("%w: could not create image", err))
			return nil, err
		}
		certImage, err = p.getImage(ctx, originalImageDigest)
		if err != nil {
			log.Error(fmt.Errorf("%w: could not get image", err))
			return nil, err
		}
	}

	// Create the RPM manifest, or get it if it already exists.
	rpmManifest := certInput.RpmManifest
	rpmManifest.ImageID = certImage.ID
	_, err = p.createRPMManifest(ctx, rpmManifest)
	if err != nil {
		if err != errors.ErrPyxis409StatusCode {
			log.Error(fmt.Errorf("%w: could not create rpm manifest", err))
			return nil, err
		}
		_, err = p.getRPMManifest(ctx, rpmManifest.ImageID)
		if err != nil {
			log.Error(fmt.Errorf("%w: could not get rpm manifest", err))
			return nil, err
		}
	}

	// Create the artifacts in Pyxis.
	artifacts := certInput.Artifacts
	for _, artifact := range artifacts {
		artifact.ImageID = certImage.ID
		if _, err := p.createArtifact(ctx, &artifact); err != nil {
			log.Error(fmt.Errorf("%w: could not create artifact: %s", err, artifact.Filename))
			return nil, err
		}
	}

	// Create the test results.
	testResults := certInput.TestResults
	testResults.ImageID = certImage.ID
	testResults, err = p.createTestResults(ctx, testResults)
	if err != nil {
		log.Error(fmt.Errorf("%w: could not create test results", err))
		return nil, err
	}

	// Return the results with up-to-date information.
	return &CertificationResults{
		CertProject: certProject,
		CertImage:   certImage,
		TestResults: testResults,
	}, nil
}
