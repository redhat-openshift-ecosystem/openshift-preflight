.DEFAULT_GOAL:=help

BINARY=preflight
IMAGE_BUILDER?=podman
IMAGE_REPO?=quay.io/opdev
VERSION=$(shell git rev-parse HEAD)
RELEASE_TAG ?= "0.0.0"

PLATFORMS=linux
ARCHITECTURES=amd64 arm64 ppc64le s390x

.PHONY: build
build:
	go build -o $(BINARY) -ldflags "-X github.com/redhat-openshift-ecosystem/openshift-preflight/version.commit=$(VERSION) -X github.com/redhat-openshift-ecosystem/openshift-preflight/version.version=$(RELEASE_TAG)" main.go
	@ls | grep preflight

.PHONY: build-multi-arch
build-multi-arch:
	$(foreach GOOS, $(PLATFORMS),\
	$(foreach GOARCH, $(ARCHITECTURES),\
	$(shell export GOOS=$(GOOS); export GOARCH=$(GOARCH);go build -o $(BINARY)-$(GOOS)-$(GOARCH) -ldflags "-X github.com/redhat-openshift-ecosystem/openshift-preflight/version.commit=$(VERSION) -X github.com/redhat-openshift-ecosystem/openshift-preflight/version.version=$(RELEASE_TAG)" main.go)))
	@ls | grep preflight

.PHONY: fmt
fmt:
	go fmt ./...
	git diff --exit-code

.PHONY: tidy
tidy:
	go mod tidy
	git diff --exit-code

.PHONY: image-build
image-build:
	$(IMAGE_BUILDER) build -t $(IMAGE_REPO)/preflight:$(VERSION) .

.PHONY: image-push
image-push:
	$(IMAGE_BUILDER) push $(IMAGE_REPO)/preflight:$(VERSION)

.PHONY: test
test:
	go test -v $$(go list ./... | grep -v e2e)

.PHONY: vet
vet:
	go vet ./...

.PHONY: test-e2e
test-e2e:
	./test/e2e/operator-test.sh

.PHONY: clean
clean:
	@go clean
	@# cleans the binary created by make build
	$(shell if [ -f "$(BINARY)" ]; then rm -f $(BINARY); fi)
	@# cleans all the binaries created by make build-multi-arch
	$(foreach GOOS, $(PLATFORMS),\
	$(foreach GOARCH, $(ARCHITECTURES),\
	$(shell if [ -f "$(BINARY)-$(GOOS)-$(GOARCH)" ]; then rm -f $(BINARY)-$(GOOS)-$(GOARCH); fi)))
