.DEFAULT_GOAL:=help

VERSION=$(shell git rev-parse HEAD)

.PHONY: build
build:
	go build -o preflight -ldflags "-X github.com/komish/preflight/version.commit=$(VERSION)" main.go

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
