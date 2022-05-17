GO := go

ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
CACHE_DIR := $(ROOT_DIR)/.cache

export GOBIN := $(ROOT_DIR)/.gobin
export PATH := $(GOBIN):$(PATH)

GORELEASER := $(GO) run github.com/goreleaser/goreleaser@v1.7.0
GOLANGCI_LINT := $(GO) run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.45.2

GOTESTSUM := $(GO) run gotest.tools/gotestsum@v1.7.0

GOREADME := $(GO) run github.com/posener/goreadme/cmd/goreadme@v1.4.2

.PHONY: readme
readme:
	$(GOREADME) \
		-constants \
		-credit=false \
		-types \
		-methods \
		-variabless \
		> README.md

.PHONY: setup-githooks
setup-githooks:
	git config core.hooksPath githooks

.PHONY: test
test:
	$(GOTESTSUM) \
		--junitfile report.xml \
		-- \
		-race \
		-cover \
		-coverprofile=coverage.out \
		-tags=integration \
		./... \
		-timeout=120m

.PHONY: coverage
coverage: test
	$(GO) tool cover -html coverage.out

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run \
		--enable-all \
		--disable=godox,varnamelen,exhaustivestruct \
		--timeout 10m

.PHONY: tidy
tidy:
	$(GO) mod tidy
	$(GO) mod vendor

.PHONY: build
build:
	$(GORELEASER) build --rm-dist --snapshot

.PHONY: release
release:
	$(GORELEASER) release --rm-dist

.PHONY: release-snapshot
release-snapshot:
	$(GORELEASER) release --rm-dist --snapshot --skip-publish
