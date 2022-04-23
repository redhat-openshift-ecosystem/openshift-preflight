package container

import (
	"time"
)

var (
	checkContainerTimeout time.Duration = 10 * time.Second
	waitContainer         time.Duration = 2 * time.Second
	certDocumentationURL                = "https://access.redhat.com/documentation/en-us/red_hat_openshift_certification/4.9/html-single/red_hat_openshift_software_certification_policy_guide/index#assembly-requirements-for-container-images_openshift-sw-cert-policy-introduction"
)
