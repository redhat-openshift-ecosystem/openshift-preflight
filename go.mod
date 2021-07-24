module github.com/redhat-openshift-ecosystem/openshift-preflight

go 1.16

require (
	github.com/containers/podman/v3 v3.2.2
	github.com/docker/docker v20.10.7+incompatible
	github.com/google/go-containerregistry v0.6.0
	github.com/knqyf263/go-rpmdb v0.0.0-20201215100354-a9e3110d8ee1
	github.com/manifoldco/promptui v0.8.0
	github.com/mattn/go-isatty v0.0.13 // indirect
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.2-0.20190823105129-775207bd45b6
	github.com/operator-framework/api v0.10.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.8.1
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.3
)
