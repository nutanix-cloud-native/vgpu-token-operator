RELEASE_NAME := $(GITHUB_REPOSITORY)
NAMESPACE := vgpu-system
CHART_DIR := charts/$(RELEASE_NAME)

.PHONY: helm-install
helm-install: helm-dependencies
	helm upgrade --install $(RELEASE_NAME) $(CHART_DIR) \
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
helm-lint:
	./hack/lint-helm.sh
