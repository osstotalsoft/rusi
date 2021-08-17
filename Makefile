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
	GOOS=linux go build -o $(OUT_DIR) -ldflags "-s -w" ./cmd/rusid ./cmd/injector

run-protoc:
	protoc proto/runtime/v1/* --go_out=.

################################################################################
# Target: docker                                                               #
################################################################################
include docker/docker.mk

