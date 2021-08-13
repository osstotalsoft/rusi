# ------------------------------------------------------------
# Copyright (c) Microsoft Corporation and Dapr Contributors.
# Licensed under the MIT License.
# ------------------------------------------------------------


# Docker image build and push setting
DOCKER:=docker
DOCKERFILE_DIR?=./docker

RUSI_SYSTEM_IMAGE_NAME=$(RELEASE_NAME)
RUSI_RUNTIME_IMAGE_NAME=rusid

# build docker image for linux
BIN_PATH=$(OUT_DIR)
DOCKERFILE:=Dockerfile
BIN_PATH := $(BIN_PATH)/release

check-docker-env:
ifeq ($(RUSI_REGISTRY),)
	$(error RUSI_REGISTRY environment variable must be set)
endif
ifeq ($(RUSI_TAG),)
	$(error RUSI_TAG environment variable must be set)
endif

docker-build: check-docker-env
	$(DOCKER) build -f $(DOCKERFILE_DIR)/$(DOCKERFILE) $(DOCKERFILE_DIR)/. -t $(RUSI_REGISTRY)/$(RUSI_SYSTEM_IMAGE_NAME):$(RUSI_TAG)

docker-push: check-docker-env
	$(DOCKER) push $(RUSI_REGISTRY)/$(RUSI_SYSTEM_IMAGE_NAME):$(RUSI_TAG)
