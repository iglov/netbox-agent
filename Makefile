PKG_PREFIX := github.com/iglov/netbox-agent
NAME := netbox-agent
BINDIR := bin
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT_TAG := $(shell git rev-parse --short=12 HEAD)
DATEINFO_TAG := $(shell date -u +'%Y%m%d%H%M%S')

PKG_TAG ?= $(shell git tag -l --points-at HEAD)
ifeq ($(PKG_TAG),)
PKG_TAG := $(BRANCH)
endif

LDFLAGS = -X 'main.Version=$(PKG_TAG)-$(DATEINFO_TAG)-$(COMMIT_TAG)'
GOBUILD = go build -trimpath -tags "$(BUILDTAGS)" -ldflags "$(LDFLAGS)"

PLATFORM_LIST = \
    linux-amd64 \

all: check test build

clean:
		rm -rf bin/*

build: linux-amd64

fmt:
		gofmt -l -w -s ./

vet:
		go vet ./...

lint: install-golint
		golint ./...

install-golint:
		which golint || go install golang.org/x/lint/golint@latest

govulncheck: install-govulncheck
		govulncheck ./...

install-govulncheck:
		which govulncheck || go install golang.org/x/vuln/cmd/govulncheck@latest

errcheck: install-errcheck
		errcheck -exclude=errcheck_excludes.txt ./...

install-errcheck:
		which errcheck || go install github.com/kisielk/errcheck@latest

golangci-lint: install-golangci-lint
		golangci-lint run --exclude '(SA4003|SA1019|SA5011):' -E contextcheck -E decorder --timeout 2m

install-golangci-lint:
		which golangci-lint || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.60.1

check: fmt vet lint errcheck golangci-lint govulncheck

test-main:
		go test ./...

test-race:
		go test -race ./...

test-pure:
		CGO_ENABLED=0 go test ./...

test-full:
		go test -coverprofile=coverage.txt -covermode=atomic ./...

benchmark:
		go test -bench=. ./...

benchmark-pure:
		CGO_ENABLED=0 go test -bench=. ./...

test: test-main test-race test-pure test-full benchmark benchmark-pure

linux-amd64:
		CGO_ENABLED=0 GOARCH=amd64 GOOS=linux $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

gz_releases=$(addsuffix .gz, $(PLATFORM_LIST))

$(gz_releases): %.gz : %
		chmod +x $(BINDIR)/$(NAME)-$(basename $@)
		gzip -f -S -$(PKG_TAG).gz $(BINDIR)/$(NAME)-$(basename $@)

all-arch: $(PLATFORM_LIST)

release: $(gz_releases)

