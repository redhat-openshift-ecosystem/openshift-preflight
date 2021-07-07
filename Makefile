.DEFAULT_GOAL:=help

VERSION=$(shell git rev-parse HEAD)
RELEASE_TAG ?= "0.0.0"

.PHONY: build
build:
	go build -o preflight -ldflags "-X github.com/redhat-openshift-ecosystem/openshift-preflight/version.commit=$(VERSION) -X github.com/redhat-openshift-ecosystem/openshift-preflight/version.version=$(RELEASE_TAG)" main.go

.PHONY: fmt
fmt:
	go fmt ./...
	git diff --exit-code

.PHONY: test
test:
	go test -v ./... 

.PHONY: vet
vet:
	go vet ./...
