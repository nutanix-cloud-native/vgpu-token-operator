# Copyright 2025 Nutanix. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

GORELEASER_PARALLELISM ?= $(shell nproc --ignore=1)
GORELEASER_VERBOSE ?= false
GORELEASER_PUSH_SNAPSHOT_IMAGES ?= false # GORELEASER_PUSH_SNAPSHOT_IMAGES is a variable used to determine whethre or not to push to nutanix harbor
export OCI_REPOSITORY ?= harbor.eng.nutanix.com/nkp

ifndef GORELEASER_CURRENT_TAG
export GORELEASER_CURRENT_TAG=$(GIT_TAG)
endif

.PHONY: build-snapshot
build-snapshot: ## Builds a snapshot with goreleaser
build-snapshot: go-generate ; $(info $(M) building snapshot $*)
	goreleaser --verbose=$(GORELEASER_VERBOSE) \
		build \
		--snapshot \
		--clean \
		--parallelism=$(GORELEASER_PARALLELISM) \
		--config=<(env GOOS=$(shell go env GOOS) gojq --yaml-input --yaml-output '.builds[0].goos |= (. + [env.GOOS] | unique)' .goreleaser.yaml) \
		$(if $(BUILD_ALL),,--single-target)

.PHONY: release-snapshot
release-snapshot: ## Builds a snapshot release with goreleaser
release-snapshot: go-generate ; $(info $(M) building snapshot release $*)
	GORELEASER_PUSH_SNAPSHOT_IMAGES=$(GORELEASER_PUSH_SNAPSHOT_IMAGES) \
	goreleaser --verbose=$(GORELEASER_VERBOSE) \
		release \
		--snapshot \
		--clean \
		--parallelism=$(GORELEASER_PARALLELISM) \
		--timeout=60m

.PHONY: release
release: ## Builds a snapshot release with goreleaser
release: go-generate ; $(info $(M) release $*)
	goreleaser --verbose=$(GORELEASER_VERBOSE) \
		release \
		--clean \
		--parallelism=$(GORELEASER_PARALLELISM) \
		--timeout=60m \
		$(GORELEASER_FLAGS)
