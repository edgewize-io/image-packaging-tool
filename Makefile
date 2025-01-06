LATEST_TAG:=$(shell git rev-list --tags --max-count=1)
ifneq (, $(LATEST_TAG))
TOOL_VERSION = $(shell git describe --tags $(LATEST_TAG) )
else
TOOL_VERSION = latest
endif


GITVERSION:=$(shell git --version | grep ^git | sed 's/^.* //g')
GITCOMMIT:=$(shell git rev-parse HEAD)

BUILD_TARGET=target
TARGET_TOOL_WITH_VERSION=packctl-$(TOOL_VERSION)

GO_ENV=CGO_ENABLED=0
GO_MODULE=GO111MODULE=on
VERSION_PKG=github.com/edgewize-io/image-packaging-tool
GO_FLAGS=-ldflags="-X ${VERSION_PKG}.Version=$(TOOL_VERSION) -X ${VERSION_PKG}.GitRevision=$(GITCOMMIT) -X ${VERSION_PKG}.BuildDate=$(shell date -u +'%Y-%m-%d')"
GO=go

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif


build:
	rm -rf $(BUILD_TARGET)/local/$(TARGET_TOOL_WITH_VERSION)
	$(GO) build $(GO_FLAGS) -o $(BUILD_TARGET)/local/$(TARGET_TOOL_WITH_VERSION)/packctl .

build-linux-amd64:
	rm -rf $(BUILD_TARGET)/amd/$(TARGET_TOOL_WITH_VERSION)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(GO_FLAGS) -o $(BUILD_TARGET)/amd/$(TARGET_TOOL_WITH_VERSION)/packctl-amd .

build-linux-arm64:
	rm -rf $(BUILD_TARGET)/arm/$(TARGET_TOOL_WITH_VERSION)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GO) build $(GO_FLAGS) -o $(BUILD_TARGET)/arm/$(TARGET_TOOL_WITH_VERSION)/packctl-arm .

clean:
	$(GO) clean ./...
	rm -rf $(BUILD_TARGET)

test: clean build-linux-amd64
	 scp target/amd/packctl-latest/packctl root@172.31.187.201:/usr/local/bin/
	#scp target/amd/packctl-latest/packctl root@192.168.120.2:/usr/local/bin/