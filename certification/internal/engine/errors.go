package engine

import "errors"

var (
	ErrGetRemoteContainerFailed = errors.New("failed to pull remote container")
	ErrCreateTempDir            = errors.New("failed to create temporary directory")
	ErrExtractingTarball        = errors.New("failed to extract tarball")
	ErrInvalidImageUri          = errors.New("image uri could not be parsed")
	ErrImageInspectFailed       = errors.New("failed to inspect image")
	ErrSaveFileFailed           = errors.New("failed to save file to artifacts directory")
	ErrRPMPackageList           = errors.New("could not get rpm list")
)
