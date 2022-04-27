package service

import "context"

type TagLister interface {
	ListTags(ctx context.Context, imageURI string) ([]string, error)
}
