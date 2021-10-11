################################################################################
# Variables                                                                    #
################################################################################

OUT_DIR := ./dist
BINARIES ?= rusid injector

# Helm template and install setting
HELM:=helm
RELEASE_NAME?=rusi
RUSI_NAMESPACE?=rusi-system
HELM_CHART_ROOT:=./helm

################################################################################
# Target: build-linux                                                          #
################################################################################
build-linux:
	mkdir -p $(OUT_DIR)
	CGO_ENABLED=0 GOOS=linux go build -o $(OUT_DIR) -ldflags "-s -w" ./cmd/rusid ./cmd/injector

modtidy:
	go mod tidy

init-proto:
	go get google.golang.org/protobuf/cmd/protoc-gen-go google.golang.org/grpc/cmd/protoc-gen-go-grpc

clean-proto:
	rm pkg/proto/runtime/v1/*.go

gen-proto:
	protoc proto/runtime/v1/* --go-grpc_out=. --go_out=. --go-grpc_opt=require_unimplemented_servers=false

upgrade-all:
	go get -u ./...
	go mod tidy

test:
	go test -race `go list ./... | grep -v 'rusi/pkg/operator'`

testV:
	go test -race -v `go list ./... | grep -v 'rusi/pkg/operator'`

include docker/docker.mk
include pkg/operator/tools/generate_kube_crd.mk
