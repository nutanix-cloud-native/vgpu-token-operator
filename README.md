<!--
 Copyright 2025 Nutanix. All rights reserved.
 SPDX-License-Identifier: Apache-2.0
 -->

# VGPU Token Operator

Kubernetes native way to handle VGPU tokens and license validation.

## Development

Install tools

- [Devbox](https://github.com/jetpack-io/devbox?tab=readme-ov-file#installing-devbox)
- [Direnv](https://direnv.net/docs/installation.html)
- Container Runtime for your Operating System

### Deploy operator

To deploy the operator during development you will need the following:

#### Prerequisites

1. VGPU drivers on host hyper-visor
1. VM Image with VGPU driver installed
1. Kubernetes cluster
1. GPU operator deployed
1. Valid vgpu token with a corresponding license server

1. Deploy helm chart to cluster

```bash
make helm-install-snapshot
```

_NOTE_: set `OCI_REPOSITORY`to a repository that you have push access to

1. Create manifests

```bash
kubectl apply -f hack/examples/vgpu.yaml
```
