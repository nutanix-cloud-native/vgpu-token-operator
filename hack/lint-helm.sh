#!/usr/bin/env bash
set -exou pipefail

CT_CONFIG="charts/ct-config.yaml"
export KIND_CLUSTER_NAME="chart-testing"
IMAGE_REPO="ko.local/vgpu-token-operator"
COPY_IMAGE_REPO="ghcr.io/nutanix-cloud-native/vgpu-token-copier"
VERSION=$(gojq -r .version dist/metadata.json)
ARCH=$(go env GOARCH)
IMAGE_TAG="${VERSION}-${ARCH}"

CLEANUP_CLUSTER=0

function cleanup() {
  if [[ "${CLEANUP_CLUSTER}" -eq 1 ]]; then
    echo "Cleaning up KinD cluster..."
    make kind.delete
  fi
}

trap cleanup EXIT

echo "Checking for changed charts..."
changed_charts=$(ct list-changed --config "$CT_CONFIG")

if [[ -z "$changed_charts" ]]; then
  echo "No charts changed - exiting"
  exit 0
fi

echo "Changed charts detected:"
echo "$changed_charts"

echo "Linting changed charts..."
ct lint --config "$CT_CONFIG"

echo "Creating KinD cluster..."
make kind.create
CLEANUP_CLUSTER=1

echo "Building Docker images..."
make release-snapshot

eval $(make kind.kubeconfig)
echo "Loading image into KinD cluster..."
kind load docker-image \
  --name "${KIND_CLUSTER_NAME}" \
  "${IMAGE_REPO}:v${IMAGE_TAG}"
kind load docker-image \
  --name "${KIND_CLUSTER_NAME}" \
  "${COPY_IMAGE_REPO}:v${IMAGE_TAG}"

ct install \
  --config "$CT_CONFIG" \
  --helm-extra-set-args "--set-string controllerManager.container.image.repository=${IMAGE_REPO} --set-string controllerManager.container.image.tag=v${IMAGE_TAG} --set-string vgpuCopy.image=${COPY_IMAGE_REPO}:v${IMAGE_TAG}"

echo "Chart testing completed successfully!"
