# Copyright 2025 Nutanix. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

# Image URL to use all building/pushing image targets
IMG ?= controller:latest

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

REPO_ROOT := $(CURDIR)
include make/all.mk

.PHONY: all
all: build

.PHONY: go-generate
go-generate:
	controller-gen paths="./internal/controller/..."  \
		rbac:headerFile="hack/license-header.yaml.txt",roleName=vgpu-token-operator-manager-role \
		output:rbac:artifacts:config=charts/vgpu-token-operator/templates/rbac
	controller-gen paths="./api/v1alpha1"  \
	  object:headerFile="hack/license-header.go.txt" output:object:artifacts:config=/api/v1alpha1 \
	  crd:headerFile=hack/license-header.yaml.txt output:crd:artifacts:config=charts/vgpu-token-operator/templates/crd
.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

# TODO(user): To use a different vendor for e2e tests, modify the setup under 'tests/e2e'.
# The default setup assumes Kind is pre-installed and builds/loads the Manager Docker image locally.
# CertManager is installed by default; skip with:
# - CERT_MANAGER_INSTALL_SKIP=true
.PHONY: test-e2e
test-e2e: go-generate fmt vet ## Run the e2e tests. Expected an isolated environment using Kind.
	@$(KIND) get clusters | grep -q 'kind' || { \
		echo "No Kind cluster is running. Please start a Kind cluster before running the e2e tests."; \
		exit 1; \
	}
	go test ./test/e2e/ -v -ginkgo.v

.PHONY: lint
lint:
	golangci-lint run --fix


.PHONY: test
test: go-generate
	source <(setup-envtest use -p env $(ENVTEST_VERSION)) && \
	gotestsum \
		--jsonfile test.json \
		--junitfile junit-report.xml \
		--junitfile-testsuite-name=relative \
		--junitfile-testcase-classname=short \
		-- \
		-covermode=atomic \
		-coverprofile=coverage.out \
		-short \
		-race \
		-v \
		$$(go list ./... | grep -v /e2e) && \
	go tool cover \
		-html=coverage.out \
		-o coverage.html
