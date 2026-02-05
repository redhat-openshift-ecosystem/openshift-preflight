package test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
)

func NewTestLoggerContext(ctx context.Context) context.Context {
	log := funcr.New(func(prefix, args string) {
		GinkgoWriter.Println(prefix, args)
	}, funcr.Options{})
	return logr.NewContext(ctx, log)
}
