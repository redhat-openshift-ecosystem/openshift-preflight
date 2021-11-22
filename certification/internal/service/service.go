package service

type TagLister interface {
	ListTags(imageURI string) ([]string, error)
}
