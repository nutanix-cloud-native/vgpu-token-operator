GORELEASER_PARALLELISM ?= $(shell nproc --ignore=1)
GORELEASER_VERBOSE ?= false

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
