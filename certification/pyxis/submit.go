package pyxis

import (
	"context"

	log "github.com/sirupsen/logrus"
)

func (p *pyxisEngine) SubmitResults(certProject *CertProject) (*CertProject, *CertImage, error) {
	var err error
	ctx := context.Background()
	oldProject := certProject

	if certProject.CertificationStatus == "Started" {
		certProject.CertificationStatus = "In Progress"
	}

	if certProject != oldProject {
		certProject, err = p.updateProject(ctx, p.ProjectId, certProject)
		if err != nil {
			log.Error(err, "could not update project")
			return nil, nil, err
		}
	}

	certImage := new(CertImage)
	certImage, err = p.createImage(ctx, certImage)
	if err != nil {
		log.Error(err)
		return nil, nil, err
	}

	rpms := make([]RPM, 10)
	err = p.createRPMManifest(ctx, certImage.ImageID, rpms)
	if err != nil {
		log.Error(err)
		return nil, nil, err
	}

	return certProject, certImage, nil
}
