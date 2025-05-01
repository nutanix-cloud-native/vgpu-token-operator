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
