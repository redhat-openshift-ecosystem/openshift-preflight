package authn

import (
	"os"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/types"
	craneauthn "github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type preflightKeychain struct{}

var PreflightKeychain craneauthn.Keychain = &preflightKeychain{}

// Resolve returns an Authenticator with credentials, or Anonymous if no suitable credentials
// are found for the target. This implements the Keychain interface from go-containerregistry,
// and will be passed to crane,.
//
// If the viper config is empty, assume Anonymous.
// If the file cannot be found or read, that constitues an error.
func (k *preflightKeychain) Resolve(target craneauthn.Resource) (craneauthn.Authenticator, error) {
	log.Trace("entering preflight keychain Resolve")

	configFile := viper.GetString("dockerConfig")
	if configFile == "" {
		// No file specified. No auth expected
		return craneauthn.Anonymous, nil
	}

	r, err := os.Open(configFile)
	if os.IsNotExist(err) {
		log.Errorf("could not find authfile: %s", configFile)
		return nil, err
	}
	if err != nil {
		log.Errorf("Could not open authfile: %s", configFile)
		return nil, err
	}
	defer r.Close()
	cf, err := config.LoadFromReader(r)
	if err != nil {
		log.Errorf("Could not load authfile: %s", configFile)
		return nil, err
	}

	var cfg, empty types.AuthConfig
	for _, key := range []string{
		target.String(),
		target.RegistryStr(),
	} {
		if key == name.DefaultRegistry {
			key = craneauthn.DefaultAuthKey
		}

		cfg, err = cf.GetAuthConfig(key)
		if err != nil {
			return nil, err
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
