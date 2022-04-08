package pyxis

import (
	"context"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	log "github.com/sirupsen/logrus"
)

// SubmitResults takes certInput and sends requests to Pyxis to create or update entries
// based on certInput.
func (p *pyxisClient) SubmitResults(ctx context.Context, certInput *certificationInput) (*CertificationResults, error) {
	var err error

	certProject := certInput.CertProject
	certImage := certInput.CertImage

	// Submission effectively starts the certification process, so switch
	// the status to reflect this if needed.
	if certProject.CertificationStatus == "Started" {
		certProject.CertificationStatus = "In Progress"
	}

	// You must have an existing repository.
	if len(certImage.Repositories) == 0 {
		return nil, errors.ErrInvalidCertImage
	}

	// Set this project's metadata to match the image that we're certifying.
	certProject.Container.Registry = certImage.Repositories[0].Registry
	certProject.Container.Repository = certImage.Repositories[0].Repository

	// Compare the original
	oldCertProject, err := p.GetProject(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, "could not retrieve project")
	}

	if *certProject != *oldCertProject {
		certProject, err = p.updateProject(ctx, certProject)
		if err != nil {
			log.Error(err, "could not update project")
			return nil, err
		}
	}

	// store the original digest so that we can pull the image later
	// in the event that it exists. createImage will wipe it otherwise.
	originalImageDigest := certImage.DockerImageDigest

	// Create the image, or get it if it already exists.
	certImage, err = p.createImage(ctx, certImage)
	if err != nil && err != errors.ErrPyxis409StatusCode {
		log.Error(err, "could not create image")
		return nil, err
	}

	if err != nil && err == errors.ErrPyxis409StatusCode {
		certImage, err = p.getImage(ctx, originalImageDigest)
		if err != nil {
			log.Error(err, "could not get image")
			return nil, err
		}
	}

	// Create the RPM manifest, or get it if it already exists.
	rpmManifest := certInput.RpmManifest
	rpmManifest.ImageID = certImage.ID
	_, err = p.createRPMManifest(ctx, rpmManifest)
	if err != nil && err != errors.ErrPyxis409StatusCode {
		log.Error(err, "could not create rpm manifest")
		return nil, err
	}
	if err != nil && err == errors.ErrPyxis409StatusCode {
		_, err = p.getRPMManifest(ctx, rpmManifest.ImageID)
		if err != nil {
			log.Error(err, "could not get rpm manifest")
			return nil, err
		}
	}

	// Create the artifacts in Pyxis.
	artifacts := certInput.Artifacts
	for _, artifact := range artifacts {
		artifact.ImageID = certImage.ID
		if _, err := p.createArtifact(ctx, &artifact); err != nil {
			log.Errorf("%s: could not create artifact: %s", err, artifact.Filename)
			return nil, err
		}
	}

	// Create the test results.
	testResults := certInput.TestResults
	testResults.ImageID = certImage.ID
	testResults, err = p.createTestResults(ctx, testResults)
	if err != nil {
		log.Error(err, "could not create test results")
		return nil, err
	}

	// Return the results with up-to-date information.
	return &CertificationResults{
		CertProject: certProject,
		CertImage:   certImage,
		TestResults: testResults,
	}, nil
}
