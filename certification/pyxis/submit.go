package pyxis

import (
	"context"

	log "github.com/sirupsen/logrus"
)

func (p *pyxisEngine) SubmitResults(certProject *CertProject, certImage *CertImage, rpmManifest *RPMManifest, testResults *TestResults) (*CertProject, *CertImage, *TestResults, error) {
	var err error
	ctx := context.Background()
	oldProject := certProject

	if certProject.CertificationStatus == "Started" {
		certProject.CertificationStatus = "In Progress"
	}

	if certProject != oldProject {
		certProject, err = p.updateProject(ctx, certProject)
		if err != nil {
			log.Error(err, "could not update project")
			return nil, nil, nil, err
		}
	}

	certImage, err = p.createImage(ctx, certImage)
	if err != nil {
		log.Error(err, "could not create image")
		return nil, nil, nil, err
	}

	rpmManifest.ImageID = certImage.ID
	_, err = p.createRPMManifest(ctx, rpmManifest)
	if err != nil {
		log.Error(err, "could not create rpm manifest")
		return nil, nil, nil, err
	}

	testResults.ImageID = certImage.ID
	testResults, err = p.createTestResults(ctx, testResults)
	if err != nil {
		log.Error(err, "could not create test results")
		return nil, nil, nil, err
	}

	return certProject, certImage, testResults, nil
}
