package engine

import (
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
)

// AuthConfig will check the provided credentials and return
// a properly configured authentication option to be used by crane function
// calls, or nil if no auth configuration was derived.
func AuthConfig(creds RegistryCredentials) *crane.Option {
	if notEmpty([]string{creds.Username, creds.Password}...) {
		authOption := crane.WithAuth(&authn.Basic{
			Username: creds.Username,
			Password: creds.Password,
		})

		return &authOption
	}

	return nil
}

// notEmpty returns true if all strings in s are non-zero length strings
func notEmpty(s ...string) bool {
	for _, v := range s {
		if len(v) <= 0 {
			return false
		}
	}

	return true
}
