// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package authn

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	craneauthn "github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	fresh              = 0
	testRegistry, _    = name.NewRegistry("test.io", name.WeakValidation)
	testRepo, _        = name.NewRepository("test.io/my-repo", name.WeakValidation)
	defaultRegistry, _ = name.NewRegistry(name.DefaultRegistry, name.WeakValidation)
)

// setupConfigDir sets up an isolated configDir() for this test.
func setupConfigDir(t *testing.T) string {
	tmpdir := os.Getenv("TEST_TMPDIR")
	if tmpdir == "" {
		var err error
		tmpdir, err = ioutil.TempDir("", "keychain_test")
		if err != nil {
			t.Fatalf("creating temp dir: %v", err)
		}
	}

	fresh++
	p := filepath.Join(tmpdir, fmt.Sprintf("%d", fresh))
	if err := os.Mkdir(p, 0o777); err != nil {
		t.Fatalf("mkdir %q: %v", p, err)
	}
	return p
}

func setupConfigFile(t *testing.T, content string) string {
	cd := setupConfigDir(t)
	p := filepath.Join(cd, "config.json")
	if err := ioutil.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write %q: %v", p, err)
	}
	os.Setenv("PFLT_DOCKERCONFIG", p)
	// return the config dir so we can clean up
	return cd
}

func TestNoConfig(t *testing.T) {
	cd := setupConfigDir(t)
	defer os.RemoveAll(filepath.Dir(cd))

	auth, err := PreflightKeychain.Resolve(testRegistry)
	if err != nil {
		t.Fatalf("Resolve() = %v", err)
	}

	if auth != craneauthn.Anonymous {
		t.Errorf("expected Anonymous, got %v", auth)
	}
}

func encode(user, pass string) string {
	delimited := fmt.Sprintf("%s:%s", user, pass)
	return base64.StdEncoding.EncodeToString([]byte(delimited))
}

func TestVariousPaths(t *testing.T) {
	tests := []struct {
		desc    string
		content string
		wantErr bool
		target  craneauthn.Resource
		cfg     *craneauthn.AuthConfig
	}{{
		desc:    "invalid config file",
		target:  testRegistry,
		content: `}{`,
		wantErr: true,
	}, {
		desc:    "creds store does not exist",
		target:  testRegistry,
		content: `{"credsStore":"#definitely-does-not-exist"}`,
		wantErr: true,
	}, {
		desc:    "valid config file",
		target:  testRegistry,
		content: fmt.Sprintf(`{"auths": {"test.io": {"auth": %q}}}`, encode("foo", "bar")),
		cfg: &craneauthn.AuthConfig{
			Username: "foo",
			Password: "bar",
		},
	}, {
		desc:    "valid config file; default registry",
		target:  defaultRegistry,
		content: fmt.Sprintf(`{"auths": {"%s": {"auth": %q}}}`, craneauthn.DefaultAuthKey, encode("foo", "bar")),
		cfg: &craneauthn.AuthConfig{
			Username: "foo",
			Password: "bar",
		},
	}, {
		desc:   "valid config file; matches registry w/ v1",
		target: testRegistry,
		content: fmt.Sprintf(`{
	  "auths": {
		"http://test.io/v1/": {"auth": %q}
	  }
	}`, encode("baz", "quux")),
		cfg: &craneauthn.AuthConfig{
			Username: "baz",
			Password: "quux",
		},
	}, {
		desc:   "valid config file; matches registry w/ v2",
		target: testRegistry,
		content: fmt.Sprintf(`{
	  "auths": {
		"http://test.io/v2/": {"auth": %q}
	  }
	}`, encode("baz", "quux")),
		cfg: &craneauthn.AuthConfig{
			Username: "baz",
			Password: "quux",
		},
	}, {
		desc:   "valid config file; matches repo",
		target: testRepo,
		content: fmt.Sprintf(`{
  "auths": {
    "test.io/my-repo": {"auth": %q},
    "test.io/another-repo": {"auth": %q},
    "test.io": {"auth": %q}
  }
}`, encode("foo", "bar"), encode("bar", "baz"), encode("baz", "quux")),
		cfg: &craneauthn.AuthConfig{
			Username: "foo",
			Password: "bar",
		},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			cd := setupConfigFile(t, test.content)
			// For some reason, these tempdirs don't get cleaned up.
			defer os.RemoveAll(filepath.Dir(cd))

			auth, err := PreflightKeychain.Resolve(test.target)
			if test.wantErr {
				if err == nil {
					t.Fatal("wanted err, got nil")
				} else if err != nil {
					// success
					return
				}
			}
			if err != nil {
				t.Fatalf("wanted nil, got err: %v", err)
			}
			cfg, err := auth.Authorization()
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(cfg, test.cfg) {
				t.Errorf("got %+v, want %+v", cfg, test.cfg)
			}
		})
	}
}

type helper struct {
	u, p string
	err  error
}

func (h helper) Get(serverURL string) (string, string, error) {
	if serverURL != "example.com" {
		return "", "", fmt.Errorf("unexpected serverURL: %s", serverURL)
	}
	return h.u, h.p, h.err
}

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.TraceLevel)

	viper.SetEnvPrefix("pflt")
	viper.AutomaticEnv()
}
