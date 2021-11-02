################################################################################
# Variables                                                                    #
################################################################################

OUT_DIR := ./dist
BINARIES ?= rusid injector operator
GIT_COMMIT  = $(shell git rev-list -1 HEAD)
GIT_VERSION = $(shell git describe --always --abbrev=7 --dirty)
RUSI_VERSION = edge

# Helm template and install setting
HELM:=helm
RELEASE_NAME?=rusi
RUSI_NAMESPACE?=rusi-system
HELM_CHART_ROOT:=./helm

################################################################################
# Go build details                                                             #
################################################################################
BASE_PACKAGE_NAME := rusi

DEFAULT_LDFLAGS:=-X $(BASE_PACKAGE_NAME)/internal/version.gitcommit=$(GIT_COMMIT) \
  -X $(BASE_PACKAGE_NAME)/internal/version.gitversion=$(GIT_VERSION) \
  -X $(BASE_PACKAGE_NAME)/internal/version.version=$(RUSI_VERSION)

################################################################################
# Target: build-linux                                                          #
################################################################################
build-linux:
	mkdir -p $(OUT_DIR)
	CGO_ENABLED=0 GOOS=linux go build -o $(OUT_DIR) -ldflags "$(DEFAULT_LDFLAGS) -s -w" ./cmd/rusid ./cmd/injector ./cmd/operator

modtidy:
	go mod tidy

init-proto:
	go get google.golang.org/protobuf/cmd/protoc-gen-go google.golang.org/grpc/cmd/protoc-gen-go-grpc

clean-proto:
	rm -rf pkg/proto/*

gen-proto:
	protoc proto/runtime/v1/* --go-grpc_out=. --go_out=. --go-grpc_opt=require_unimplemented_servers=false
	protoc proto/operator/v1/* --go-grpc_out=. --go_out=. --go-grpc_opt=require_unimplemented_servers=false

upgrade-all:
	go get -u ./...
	go mod tidy

test:
	go test -race `go list ./... | grep -v 'rusi/pkg/operator'`

testV:
	go test -race -v `go list ./... | grep -v 'rusi/pkg/operator'`

include docker/docker.mk
include pkg/operator/tools/generate_kube_crd.mk
