<!--
 Copyright 2025 Nutanix. All rights reserved.
 SPDX-License-Identifier: Apache-2.0
 -->

# VGPU Token Operator

Kubernetes native way to handle VGPU tokens and license validation.

## Motivation

NVIDIA vGPUs fetch licenses through JWTs via the gridd service.
While the [NVIDIA GPU Operator](https://github.com/NVIDIA/gpu-operator/) facilitates licensing for its own driver deployments via mounted ConfigMaps, it doesn't solve the problem for systems with pre-installed vGPU drivers.
In these environments, there's no inherent mechanism to automatically update expired JWTs, forcing manual intervention on every node. This project addresses this specific gap by offering a Kubernetes-native solution to streamline the token update process for these pre-installed driver setups.

## Development

Install tools

- [Devbox](https://github.com/jetpack-io/devbox?tab=readme-ov-file#installing-devbox)
- [Direnv](https://direnv.net/docs/installation.html)
- Container Runtime for your Operating System

## Deploying the operator

VGPU token operator is deployed using a helm chart which can be found in `charts/`

### Prerequisites

Absolutely Required:

1. Kubernetes cluster
1. GPU operator deployed

Things you probably have if you're looking at this project.

1. Valid vgpu token with a corresponding license server
1. VGPU drivers on host hyper-visor
1. VM Image with VGPU driver installed

To deploy helm chart to your cluster during development run

```bash
make helm-install-snapshot
```

_NOTE_: set `OCI_REPOSITORY`to a repository that you have push access to

1. Create manifests

```bash
kubectl apply -f hack/examples/vgpu.yaml
```
