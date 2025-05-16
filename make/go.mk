# Copyright 2025 Nutanix. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

ENVTEST_VERSION=1.29.x!


.PHONY: lint
lint:
	golangci-lint run --fix

.PHONY: go-generate
go-generate:
	controller-gen paths="./internal/controller/..."  \
		rbac:headerFile="hack/license-header.yaml.txt",roleName=vgpu-token-operator-manager-role \
		output:rbac:artifacts:config=charts/vgpu-token-operator/templates/rbac
	controller-gen paths="./api/v1alpha1"  \
	  object:headerFile="hack/license-header.go.txt" output:object:artifacts:config=/api/v1alpha1 \
	  crd:headerFile=hack/license-header.yaml.txt output:crd:artifacts:config=charts/vgpu-token-operator/templates/crd

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

export CAPI_VERSION = $(shell go list -m -f '{{ .Version }}' sigs.k8s.io/cluster-api)
export CAPX_VERSION ?= v1.6.1
export CAAPH_VERSION ?= v0.3.1
E2E_DRYRUN ?= false
E2E_VERBOSE ?= $(filter $(E2E_DRYRUN),true) # If dry-run, enable verbosity
E2E_PARALLEL_NODES ?= $(if $(filter $(E2E_DRYRUN),true),1,$(shell nproc --ignore=1)) # Ginkgo cannot dry-run in parallel
E2E_FLAKE_ATTEMPTS ?= 1
E2E_CONF_FILE_TEMPLATE ?= $(REPO_ROOT)/test/e2e/config/nutanix.yaml.tmp
E2E_CONF_FILE ?= $(REPO_ROOT)/test/e2e/config/nutanix.yaml
export E2E_KUBERNETES_VERSION ?= $(KINDEST_IMAGE_TAG)
ARTIFACTS ?= ${REPO_ROOT}/_artifacts

E2E_DIR ?= ${REPO_ROOT}/test/e2e
CNI_PATH_CALICO ?= "${E2E_DIR}/data/cni/calico/calico.yaml"
E2E_TEMPLATES := ${E2E_DIR}/data/infrastructure-nutanix

cluster-e2e-templates-v1beta1: ## Generate cluster templates for v1beta1
	kustomize build $(E2E_TEMPLATES)/v1beta1/vgpu --load-restrictor LoadRestrictionsNone > $(E2E_TEMPLATES)/v1beta1/cluster-template-vgpu.yaml

.PHONY: test.e2e
test.e2e: cluster-e2e-templates-v1beta1 build-snapshot release-snapshot-images
		env CNI=$(CNI_PATH_CALICO) \
	  envsubst -no-unset -no-empty -i '$(E2E_CONF_FILE_TEMPLATE)' -o '$(E2E_CONF_FILE)'
	  ginkgo run \
	    --r \
	    --show-node-events \
	    --trace \
	    --randomize-all \
	    --randomize-suites \
	    --fail-on-pending \
	    --keep-going \
	    $(if $(filter $(E2E_VERBOSE),true),--vv) \
	    --covermode=atomic \
	    --coverprofile $(REPO_ROOT)/coverage-e2e.out \
	    $(if $(filter $(E2E_DRYRUN), true),--dry-run) \
	    --procs=1 \
	    --compilers=1 \
	    --flake-attempts=$(E2E_FLAKE_ATTEMPTS) \
	    $(if $(E2E_FOCUS),--focus="$(E2E_FOCUS)") \
	    $(if $(E2E_SKIP),--skip="$(E2E_SKIP)") \
	    $(if $(E2E_LABEL),--label-filter="$(E2E_LABEL)") \
	    $(E2E_GINKGO_FLAGS) \
	    --junit-report=junit-e2e.xml \
	    --json-report=report-e2e.json \
	    --tags e2e \
	    test/e2e/... -- \
	      -e2e.artifacts-folder="$(ARTIFACTS)" \
	      -e2e.config="$(E2E_CONF_FILE)" \
	      $(if $(filter $(E2E_SKIP_CLEANUP),true),-e2e.skip-resource-cleanup) \
	      -e2e.helm-chart-dir=$(REPO_ROOT)/charts/vgpu-token-operator  \
	      -e2e.helm-oci-repository=$(OCI_REPOSITORY) \
	      -e2e.helm-chart-version=v$(shell gojq -r .version dist/metadata.json)
	go tool cover \
	  -html=$(REPO_ROOT)/coverage-e2e.out \
	  -o coverage-e2e.html
