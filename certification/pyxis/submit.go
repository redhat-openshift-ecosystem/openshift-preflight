package pyxis

import (
	"context"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	log "github.com/sirupsen/logrus"
)

func (p *pyxisEngine) SubmitResults(containerImage string) (*CertProject, *CertImage, error) {
	ctx := context.Background()
	projectId := p.ProjectId
	if projectId == "" {
		return nil, nil, errors.ErrEmptyProjectID
	}
	if strings.HasPrefix(projectId, "ospid-") {
		projectId = strings.Split(projectId, "-")[1]
	}
	project, err := p.GetProject(ctx)
	if err != nil {
		log.Error(err, "could not retrieve project")
		return nil, nil, err
	}
	oldProject := project

	if project.CertificationStatus == "Started" {
		project.CertificationStatus = "In Progress"
	}

	if project != oldProject {
		project, err = p.updateProject(ctx, projectId, project)
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

	return project, certImage, nil
}
