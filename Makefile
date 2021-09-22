.DEFAULT_GOAL:=help

IMAGE_BUILDER?=podman
IMAGE_REPO?=quay.io/opdev
VERSION=$(shell git rev-parse HEAD)
RELEASE_TAG ?= "0.0.0"

.PHONY: build
build:
	go build -o preflight -ldflags "-X github.com/redhat-openshift-ecosystem/openshift-preflight/version.commit=$(VERSION) -X github.com/redhat-openshift-ecosystem/openshift-preflight/version.version=$(RELEASE_TAG)" main.go

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
	go test -v ./test/e2e/ 
