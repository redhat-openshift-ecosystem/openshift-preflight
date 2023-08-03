package option

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/authn"

	"github.com/google/go-containerregistry/pkg/crane"
	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type CraneConfig interface {
	CraneDockerConfig() string
	CranePlatform() string
	CraneInsecure() bool
}

func GenerateCraneOptions(ctx context.Context, craneConfig CraneConfig) []crane.Option {
	// prepare crane runtime options, if necessary
	options := []crane.Option{
		crane.WithContext(ctx),
		crane.WithAuthFromKeychain(
			authn.PreflightKeychain(
				ctx,
				// We configure the Preflight Keychain here.
				// In theory, we should not require further configuration
				// downstream because the PreflightKeychain is a singleton.
				// However, as long as we pass this same DockerConfig
				// value downstream, it shouldn't matter if the
				// keychain is reconfigured downstream.
				authn.WithDockerConfig(craneConfig.CraneDockerConfig()),
			),
		),
		crane.WithPlatform(&cranev1.Platform{
			OS:           "linux",
			Architecture: craneConfig.CranePlatform(),
		}),
		retryOnceAfter(5 * time.Second),
	}

	if craneConfig.CraneInsecure() {
		// Adding WithTransport opt is a workaround to allow for access to HTTPS
		// container registries with self-signed or non-trusted certificates.
		//
		// See https://github.com/google/go-containerregistry/issues/1553 for more context. If this issue
		// is resolved, then this workaround can likely be removed or adjusted to use new features in the
		// go-containerregistry project.
		rt := remote.DefaultTransport.(*http.Transport).Clone()
		rt.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, //nolint: gosec
		}

		options = append(options, crane.Insecure, crane.WithTransport(rt))
	}

	return options
}

// retryOnceAfter is a crane option that retries once after t duration.
func retryOnceAfter(t time.Duration) crane.Option {
	return func(o *crane.Options) {
		o.Remote = append(o.Remote, remote.WithRetryBackoff(remote.Backoff{
			Duration: t,
			Factor:   1.0,
			Jitter:   0.1,
			Steps:    2,
		}))
	}
}
