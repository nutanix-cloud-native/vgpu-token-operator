# Copyright 2025 Nutanix. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

RELEASE_NAME := $(GITHUB_REPOSITORY)
NAMESPACE ?= vgpu-system
CHART_DIR := charts/$(RELEASE_NAME)
OCI_REPOSITORY ?= harbor.eng.nutanix.com/nkp

.PHONY: helm-install-snapshot
helm-install-snapshot: export GORELEASER_PUSH_SNAPSHOT_IMAGES=true
helm-install-snapshot: release-snapshot helm-dependencies
	helm upgrade --install $(RELEASE_NAME) $(CHART_DIR) \
		--set-string controllerManager.container.image.repository=$(OCI_REPOSITORY)/vgpu-token-operator \
		--set-string controllerManager.container.image.tag="v$(shell gojq -r .version dist/metadata.json)"  \
		--set-string vgpuCopy.repository="${OCI_REPOSITORY}/vgpu-token-copier" \
		--set-string vgpuCopy.tag="v$(shell gojq -r .version dist/metadata.json)-amd64"  \
		--namespace $(NAMESPACE) \
		--create-namespace \
		--wait

.PHONY: helm-uninstall
helm-uninstall:  ## Uninstall the Helm chart
	helm uninstall $(RELEASE_NAME) --namespace $(NAMESPACE)

.PHONY: helm-dependencies
helm-dependencies:  ## Update Helm dependencies
	helm dependency update $(CHART_DIR)

.PHONY: helm-lint
helm-lint: export OCI_REPOSITORY=ghcr.io/nutanix-cloud-native
helm-lint: build-snapshot
	./hack/lint-helm.sh
