package authn

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/types"
	"github.com/go-logr/logr"
	craneauthn "github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"
)

type preflightKeychain struct {
	dockercfg string
	ctx       context.Context
}

type PreflightKeychainOption func(*preflightKeychain)

// WithDockerConfig configures the PreflightKeychain with the specified
// docker config at path dockercfg. To unset any existing dockercfg, pass
// this option with an empty string value.
func WithDockerConfig(dockercfg string) PreflightKeychainOption {
	return func(pk *preflightKeychain) {
		pk.dockercfg = dockercfg
	}
}

var keychain = preflightKeychain{
	ctx: context.Background(), // Initialize here, but can be overridden with PreflightKeychain func
}

// PreflightKeychain will return the preflight keychain as a craneauthn.Keychain.
// This operates as a singleton. If provided an option, that option overwrites
// the single instance of PreflightKeychain. If provided no option, the keychain
// is returned as already configured.
func PreflightKeychain(ctx context.Context, opts ...PreflightKeychainOption) craneauthn.Keychain {
	for _, opt := range opts {
		opt(&keychain)
	}

	keychain.ctx = ctx

	return &keychain
}

// Resolve returns an Authenticator with credentials, or Anonymous if no suitable credentials
// are found for the target. This implements the Keychain interface from go-containerregistry,
// and will be passed to crane,.
//
// If the dockerConfig value is empty, assume Anonymous.
// If the file cannot be found or read, that constitutes an error.
// Can return os.IsNotExist.
func (k *preflightKeychain) Resolve(target craneauthn.Resource) (craneauthn.Authenticator, error) {
	logger := logr.FromContextOrDiscard(k.ctx)

	logger.V(log.TRC).Info("entering preflight keychain Resolve")

	if k.dockercfg == "" {
		// No file specified. No auth expected
		return craneauthn.Anonymous, nil
	}

	r, err := os.Open(k.dockercfg)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("could not find authfile: %s: %w", k.dockercfg, err)
	}
	if err != nil {
		return nil, fmt.Errorf("could not open authfile: %s: %v", k.dockercfg, err)
	}

	defer r.Close()
	cf, err := config.LoadFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("could not load authfile from reader: %v", err)
	}

	// We'll check the authconfig for creds associated with these endpoints.
	authFileTargets := []string{
		target.String(),
		target.RegistryStr(),
	}

	// If the user logged into docker.io using podman, the auth.json would
	// contain docker.io. Crane rewrites this to index.docker.io internally,
	// but the credential file does not have an entry for this, so we also
	// check for docker.io/* entries that match.
	if strings.Contains(name.DefaultRegistry, target.RegistryStr()) {
		authFileTargets = append(authFileTargets,
			strings.Replace(target.String(), name.DefaultRegistry, "docker.io", 1),
			strings.Replace(target.RegistryStr(), name.DefaultRegistry, "docker.io", 1),
		)
	}

	var cfg, empty types.AuthConfig
	for _, key := range authFileTargets {
		if key == name.DefaultRegistry {
			key = craneauthn.DefaultAuthKey
		}

		cfg, err = cf.GetAuthConfig(key)
		if err != nil {
			return nil, fmt.Errorf("could not get auth config: %v", err)
		}
		if cfg != empty {
			break
		}
	}
	if cfg == empty {
		return craneauthn.Anonymous, nil
	}

	return craneauthn.FromConfig(craneauthn.AuthConfig{
		Username:      cfg.Username,
		Password:      cfg.Password,
		Auth:          cfg.Auth,
		IdentityToken: cfg.IdentityToken,
		RegistryToken: cfg.RegistryToken,
	}), nil
}
