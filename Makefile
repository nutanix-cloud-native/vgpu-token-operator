# Copyright 2025 Nutanix. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

REPO_ROOT := $(CURDIR)
include make/all.mk

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
