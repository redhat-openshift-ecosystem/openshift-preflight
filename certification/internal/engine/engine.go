package engine

import (
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/google/go-containerregistry/pkg/crane"
	log "github.com/sirupsen/logrus"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	fileutils "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/utils/file"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
)

// CraneEngine implements a certification.CheckEngine, and leverage crane to interact with
// the container registry and target image.
type CraneEngine struct {
	// Image is what is being tested, and should contain the
	// fully addressable path (including registry, namespaces, etc)
	// to the image
	Image string
	// Checks is an array of all checks to be executed against
	// the image provided.
	Checks []certification.Check
	// RegistryCreds are the credentials used to access to registry
	// from which the image will be pulled.
	RegistryCreds RegistryCredentials
	// IsBundle is an indicator that the asset is a bundle.
	IsBundle bool

	imageRef certification.ImageReference
	results  runtime.Results
}

func (c *CraneEngine) ExecuteChecks() error {
	log.Info("target image: ", c.Image)

	// prepare crane runtime options
	options := make([]crane.Option, 0)

	// derive authentication context
	withAuth := AuthConfig(c.RegistryCreds)
	if withAuth == nil {
		log.Info("no authentication context derived from execution configuration")
	} else {
		log.Info("accessing registry with provided authentication context")
		options = append(options, *withAuth)
	}

	// pull the image and save to fs
	log.Info("pulling image from target registry")
	img, err := crane.Pull(c.Image, options...)
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrGetRemoteContainerFailed, err)
	}

	// create tmpdir to receive extracted fs
	tmpdir, err := os.MkdirTemp(os.TempDir(), "preflight-*")
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrCreateTempDir, err)
	}
	log.Debug("temporary directory is ", tmpdir)
	defer func() {
		if err := os.RemoveAll(tmpdir); err != nil {
			log.Error("unable to clean up tmpdir", tmpdir, err)
		}
	}()

	containerFSPath := path.Join(tmpdir, "fs")
	if err := os.Mkdir(containerFSPath, 0755); err != nil {
		return fmt.Errorf("%w: %s: %s", errors.ErrCreateTempDir, containerFSPath, err)
	}

	// export/flatten, and extract
	log.Debug("exporting and flattening image")
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		log.Debug("writing container filesystem to output dir", containerFSPath)
		err = crane.Export(img, w)
		if err != nil {
			// TODO: Handle this error more effectively. Right now we rely on
			// error handling in the logic to extract this export in a lower
			// line, but we should probably exit early if the export encounters
			// an error, which requires watching multiple error streams.
			log.Error("unable to export and flatten container filesystem:", err)
		}
	}()

	log.Debug("extracting container filesystem to ", containerFSPath)
	err = fileutils.Untar(containerFSPath, r)
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrExtractingTarball, err)
	}

	// store the image internals in the engine image reference to pass to validations.
	// TODO: pass this to validations when the new check interface includes it.
	c.imageRef = certification.ImageReference{
		ImageURI:    c.Image,
		ImageFSPath: containerFSPath,
		ImageInfo:   &img,
	}

	// execute checks
	log.Info("executing checks")
	for _, check := range c.Checks {
		c.results.TestedImage = c.Image

		log.Info("running check: ", check.Name())

		// run the validation
		checkStartTime := time.Now()
		checkPassed, err := check.Validate(c.imageRef)
		checkElapsedTime := time.Since(checkStartTime)

		if err != nil {
			log.WithFields(log.Fields{"result": err, "ERROR": err.Error()}).Info("check completed: ", check.Name())
			c.results.Errors = append(c.results.Errors, runtime.Result{Check: check, ElapsedTime: checkElapsedTime})
			continue
		}

		if !checkPassed {
			log.WithFields(log.Fields{"result": "FAILED"}).Info("check completed: ", check.Name())
			c.results.Failed = append(c.results.Failed, runtime.Result{Check: check, ElapsedTime: checkElapsedTime})
			continue
		}

		log.WithFields(log.Fields{"result": "PASSED"}).Info("check completed: ", check.Name())
		c.results.Passed = append(c.results.Passed, runtime.Result{Check: check, ElapsedTime: checkElapsedTime})
	}

	if len(c.results.Errors) > 0 || len(c.results.Failed) > 0 {
		c.results.PassedOverall = false
	} else {
		c.results.PassedOverall = true
	}

	// hash contents if bundle
	if c.IsBundle {
		// TODO: Implement! The current implementation of this requires a podman engine
		// and leverages shell exec calls. Leaving this out for the moment knowing we must
		// implement before we transition.
	}

	return nil
}

// Results will return the results of check execution.
func (c *CraneEngine) Results() runtime.Results {
	return c.results
}
