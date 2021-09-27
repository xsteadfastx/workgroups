.PHONY: build
build:
	goreleaser build --rm-dist --snapshot

.PHONY: release
release:
	goreleaser release --rm-dist --snapshot --skip-publish

.PHONY: generate
generate:
	go generate

.PHONY: lint
lint:
	golangci-lint run --enable-all --disable=exhaustivestruct,godox

.PHONY: test
test:
	go test -v \
		-race \
		-coverprofile coverage.out \
		./...
	go tool cover -func coverage.out

.PHONY: tidy
tidy:
	go mod tidy
	go mod vendor

.PHONY: coverage
coverage:
	gocover-cobertura < coverage.out > coverage.xml

.PHONY: install-tools
install-tools:
	go list -f '{{range .Imports}}{{.}} {{end}}' tools.go | xargs go install -v

.PHONY: readme
readme:
	goreadme \
		-constants \
		-credit=false \
		-types \
		-methods \
		-variabless \
		> README.md

.PHONY: setup-githooks
setup-githooks:
	git config core.hooksPath githooks
