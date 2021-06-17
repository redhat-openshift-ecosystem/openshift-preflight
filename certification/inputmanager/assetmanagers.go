package inputmanager

type ContainerFileManager interface {
	ExtractContainerTar(path string) (newPath string, extractionErr error)
	ReadContainerFileMetadata(path string) error
}
