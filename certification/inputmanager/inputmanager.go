package inputmanager

// TODO: Identify what interfaces we need for both pulling down images,
// and separately, parsing the image once it's on disk. Below options
// are just staged placeholders.
type ContainerFileManager interface {
	ExtractContainerTar(path string) (newPath string, extractionErr error)
	ReadContainerFileMetadata(path string) error
}

type ContainerGetter interface {
	GetContainerFromRegistry(containerLoc string) error
}
