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

_NOTE_: set `OCI_REPOSITORY`to a repository that you have push access to

```bash
make helm-install-snapshot
```

### Creating the resources

1. Create the secret

_NOTE_: It is critical that the key for the token value is `client_configuration_token.tok`. Otherwise the mounts for the daemonset will fail.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: client-config-token
  namespace: vgpu-system
stringData:
  client_configuration_token.tok: "${VGPU_TOKEN_VALUE}"
```

1. Create the VGPUToken object, setting `tokenSecretRef` to the same name as the secret created above

```yaml
apiVersion: vgpu-token.nutanix.com/v1alpha1
kind: VGPUToken
metadata:
  name: vgpu-token
  namespace: vgpu-system
spec:
  tokenSecretRef:
    name: client-config-token

```

After creating these resources the token secret should mount on the host at `/etc/nvidia/ClientConfigToken/client_configuration_token.tok`

Finally, we can verify the license status by creating a pod to verify the license status by running `nvidia-smi` and checking the license status

```yaml
apiVersion: v1
kind: Pod
metadata:
  generateName: gpu-pod-
  labels:
    test: gpu-pod
spec:
  restartPolicy: OnFailure
  containers:
  - name: gpu-pod
    image: nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0
    command: ["nvidia-smi", "-q"]
    resources:
      limits:
        nvidia.com/gpu: 1
  nodeSelector:
    "nvidia.com/gpu.present": "true"
  tolerations:
  - key: "nvidia.com/gpu"
    operator: "Exists"
    effect: "NoSchedule"
```

In the pod logs, you should see the product is Licensed.

```bash
    vGPU Software Licensed Product
        Product Name                      : NVIDIA Virtual Compute Server
        License Status                    : Licensed (Expiry: 2025-6-5 15:22:24 GMT)
```
