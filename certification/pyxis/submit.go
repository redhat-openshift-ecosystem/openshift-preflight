package pyxis

import (
	"context"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	log "github.com/sirupsen/logrus"
)

func (p *pyxisEngine) SubmitResults(ctx context.Context, certInput *CertificationInput) (*CertificationResults, error) {
	var err error

	certProject := certInput.CertProject
	certImage := certInput.CertImage

	if certProject.CertificationStatus == "Started" {
		certProject.CertificationStatus = "In Progress"
	}

	if len(certImage.Repositories) == 0 {
		return nil, errors.ErrInvalidCertImage
	}
	certProject.Container.Registry = certImage.Repositories[0].Registry
	certProject.Container.Repository = certImage.Repositories[0].Repository

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

	dockerImageDigest := certImage.DockerImageDigest

	certImage, err = p.createImage(ctx, certImage)
	if err != nil && err != errors.Err409StatusCode {
		log.Error(err, "could not create image")
		return nil, err
	}
	if err != nil && err == errors.Err409StatusCode {
		certImage, err = p.getImage(ctx, dockerImageDigest)
		if err != nil {
			log.Error(err, "could not get image")
			return nil, err
		}
	}

	rpmManifest := certInput.RpmManifest
	rpmManifest.ImageID = certImage.ID
	_, err = p.createRPMManifest(ctx, rpmManifest)
	if err != nil && err != errors.Err409StatusCode {
		log.Error(err, "could not create rpm manifest")
		return nil, err
	}
	if err != nil && err == errors.Err409StatusCode {
		_, err = p.getRPMManifest(ctx, rpmManifest.ImageID)
		if err != nil {
			log.Error(err, "could not get rpm manifest")
			return nil, err
		}
	}

	artifacts := certInput.Artifacts
	for _, artifact := range artifacts {
		artifact.ImageID = certImage.ID
		if _, err := p.createArtifact(ctx, &artifact); err != nil {
			log.Errorf("%s: could not create artifact: %s", err, artifact.Filename)
			return nil, err
		}
	}

	testResults := certInput.TestResults
	testResults.ImageID = certImage.ID
	testResults, err = p.createTestResults(ctx, testResults)
	if err != nil {
		log.Error(err, "could not create test results")
		return nil, err
	}

	return &CertificationResults{
		CertProject: certProject,
		CertImage:   certImage,
		TestResults: testResults,
	}, nil
}
