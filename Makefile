################################################################################
# Variables                                                                    #
################################################################################

OUT_DIR := ./dist
BINARIES ?= rusid injector

# Helm template and install setting
HELM:=helm
RELEASE_NAME?=rusi
DAPR_NAMESPACE?=rusi-system
HELM_CHART_ROOT:=./helm

################################################################################
# Target: build-linux                                                          #
################################################################################
build-linux:
	GOOS=linux go build -o $(OUT_DIR) -ldflags "-s -w" ./cmd/rusid ./cmd/injector

################################################################################
# Target: docker                                                               #
################################################################################
include docker/docker.mk

