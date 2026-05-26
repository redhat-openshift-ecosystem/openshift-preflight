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
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	craneauthn "github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	fresh              = 0
	testRegistry, _    = name.NewRegistry("test.io", name.WeakValidation)
	testRepo, _        = name.NewRepository("test.io/my-repo", name.WeakValidation)
	defaultRegistry, _ = name.NewRegistry(name.DefaultRegistry, name.WeakValidation)
)

// setupConfigDir sets up an isolated configDir() for this test.
func setupConfigDir() string {
	tmpdir := os.Getenv("TEST_TMPDIR")
	if tmpdir == "" {
		var err error
		tmpdir, err = os.MkdirTemp("", "keychain_test")
		Expect(err).ToNot(HaveOccurred())
	}

	fresh++
	p := filepath.Join(tmpdir, fmt.Sprintf("%d", fresh))
	Expect(os.Mkdir(p, 0o777)).To(Succeed())
	return p
}

// setupConfigFile creates a docker config.json on disk and configures
// the PreflightKeychain to use it. It returns the config directory
// for cleanup purposes.
func setupConfigFile(content string) string {
	cd := setupConfigDir()
	p := filepath.Join(cd, "config.json")
	Expect(os.WriteFile(p, []byte(content), 0o600)).To(Succeed())

	// configure the keychain with the config provided.
	keychain.dockercfg = p
	keychain.ctx = context.TODO()

	// return the config dir so we can clean up
	return cd
}

func encode(user, pass string) string {
	delimited := fmt.Sprintf("%s:%s", user, pass)
	return base64.StdEncoding.EncodeToString([]byte(delimited))
}

var _ = Describe("PreflightKeychain", func() {
	var origCfg string
	var origCtx context.Context

	BeforeEach(func() {
		origCfg = keychain.dockercfg
		origCtx = keychain.ctx
	})

	AfterEach(func() {
		keychain.dockercfg = origCfg
		keychain.ctx = origCtx
	})

	When("the authfile does not exist", func() {
		It("should return an os.ErrNotExist error", func() {
			keychain.dockercfg = "/does/not/exist/config.json"
			keychain.ctx = context.TODO()

			_, err := keychain.Resolve(testRegistry)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(os.ErrNotExist))
		})
	})

	When("the authfile cannot be read", func() {
		It("should return an error about opening the authfile", func() {
			tmpdir, err := os.MkdirTemp("", "keychain_open_error")
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(os.RemoveAll, tmpdir)

			p := filepath.Join(tmpdir, "config.json")
			Expect(os.WriteFile(p, []byte(`{}`), 0o000)).To(Succeed())

			keychain.dockercfg = p
			keychain.ctx = context.TODO()

			_, err = keychain.Resolve(testRegistry)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(SatisfyAny(
				ContainSubstring("could not open authfile"),
				ContainSubstring("could not load authfile"),
			))
		})
	})

	When("the config has no matching credentials", func() {
		It("should return Anonymous", func() {
			cd := setupConfigFile(fmt.Sprintf(`{"auths": {"other.io": {"auth": %q}}}`, encode("foo", "bar")))
			DeferCleanup(os.RemoveAll, filepath.Dir(cd))

			auth, err := keychain.Resolve(testRegistry)
			Expect(err).ToNot(HaveOccurred())
			Expect(auth).To(Equal(craneauthn.Anonymous))
		})
	})

	When("no config file is set", func() {
		It("should return Anonymous", func() {
			cd := setupConfigDir()
			DeferCleanup(os.RemoveAll, cd)

			auth, err := keychain.Resolve(testRegistry)
			Expect(err).ToNot(HaveOccurred())
			Expect(auth).To(Equal(craneauthn.Anonymous))
		})
	})

	DescribeTable("resolving credentials from various config files",
		func(content string, wantErr bool, target craneauthn.Resource, expectedCfg *craneauthn.AuthConfig) {
			cd := setupConfigFile(content)
			DeferCleanup(os.RemoveAll, filepath.Dir(cd))

			auth, err := keychain.Resolve(target)
			if wantErr {
				Expect(err).To(HaveOccurred())
				return
			}
			Expect(err).ToNot(HaveOccurred())

			cfg, err := auth.Authorization()
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).To(Equal(expectedCfg))
		},
		Entry("invalid config file",
			`}{`,
			true,
			testRegistry,
			nil,
		),
		Entry("creds store does not exist",
			`{"credsStore":"#definitely-does-not-exist"}`,
			true,
			testRegistry,
			nil,
		),
		Entry("valid config file",
			fmt.Sprintf(`{"auths": {"test.io": {"auth": %q}}}`, encode("foo", "bar")),
			false,
			testRegistry,
			&craneauthn.AuthConfig{
				Username: "foo",
				Password: "bar",
			},
		),
		Entry("valid config file; default registry",
			fmt.Sprintf(`{"auths": {"%s": {"auth": %q}}}`, craneauthn.DefaultAuthKey, encode("foo", "bar")),
			false,
			defaultRegistry,
			&craneauthn.AuthConfig{
				Username: "foo",
				Password: "bar",
			},
		),
		Entry("valid config file as written by podman; default registry",
			fmt.Sprintf(`{"auths": {"docker.io": {"auth": %q}}}`, encode("foo", "bar")),
			false,
			defaultRegistry,
			&craneauthn.AuthConfig{
				Username: "foo",
				Password: "bar",
			},
		),
		Entry("valid config file; matches registry w/ v1",
			fmt.Sprintf(`{
	  "auths": {
		"http://test.io/v1/": {"auth": %q}
	  }
	}`, encode("baz", "quux")),
			false,
			testRegistry,
			&craneauthn.AuthConfig{
				Username: "baz",
				Password: "quux",
			},
		),
		Entry("valid config file; matches registry w/ v2",
			fmt.Sprintf(`{
	  "auths": {
		"http://test.io/v2/": {"auth": %q}
	  }
	}`, encode("baz", "quux")),
			false,
			testRegistry,
			&craneauthn.AuthConfig{
				Username: "baz",
				Password: "quux",
			},
		),
		Entry("valid config file; matches repo",
			fmt.Sprintf(`{
  "auths": {
    "test.io/my-repo": {"auth": %q},
    "test.io/another-repo": {"auth": %q},
    "test.io": {"auth": %q}
  }
}`, encode("foo", "bar"), encode("bar", "baz"), encode("baz", "quux")),
			false,
			testRepo,
			&craneauthn.AuthConfig{
				Username: "foo",
				Password: "bar",
			},
		),
	)
})
