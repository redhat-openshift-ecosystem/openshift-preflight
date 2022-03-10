package pyxis

import (
	"context"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	log "github.com/sirupsen/logrus"
)

func (p *pyxisEngine) SubmitResults(ctx context.Context, certProject *CertProject, certImage *CertImage, rpmManifest *RPMManifest, testResults *TestResults) (*CertProject, *CertImage, *TestResults, error) {
	var err error

	if certProject.CertificationStatus == "Started" {
		certProject.CertificationStatus = "In Progress"
	}

	oldCertProject, err := p.GetProject(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: %s", err, "could not retrieve project")
	}

	if *certProject != *oldCertProject {
		certProject, err = p.updateProject(ctx, certProject)
		if err != nil {
			log.Error(err, "could not update project")
			return nil, nil, nil, err
		}
	}

	dockerImageDigest := certImage.DockerImageDigest

	certImage, err = p.createImage(ctx, certImage)
	if err != nil && err != errors.Err409StatusCode {
		log.Error(err, "could not create image")
		return nil, nil, nil, err
	}
	if err != nil && err == errors.Err409StatusCode {
		certImage, err = p.getImage(ctx, dockerImageDigest)
		if err != nil {
			log.Error(err, "could not get image")
			return nil, nil, nil, err
		}
	}

	rpmManifest.ImageID = certImage.ID
	_, err = p.createRPMManifest(ctx, rpmManifest)
	if err != nil && err != errors.Err409StatusCode {
		log.Error(err, "could not create rpm manifest")
		return nil, nil, nil, err
	}
	if err != nil && err == errors.Err409StatusCode {
		_, err = p.getRPMManifest(ctx, rpmManifest.ImageID)
		if err != nil {
			log.Error(err, "could not get rpm manifest")
			return nil, nil, nil, err
		}
	}

	testResults.ImageID = certImage.ID
	testResults, err = p.createTestResults(ctx, testResults)
	if err != nil {
		log.Error(err, "could not create test results")
		return nil, nil, nil, err
	}

	return certProject, certImage, testResults, nil
}
