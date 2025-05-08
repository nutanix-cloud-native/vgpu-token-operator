# VGPUToken Design

## Summary

The NVIDIA vGPU Token Manager is a Kubernetes controller that streamlines the distribution and validation of NVIDIA vGPU JWT tokens across nodes in a Kubernetes cluster. Using the operator pattern, it automatically propagates license tokens to required nodes and verifies proper licensing, while providing a monitoring mechanism to ensure NVIDIA GRID services restart when tokens change. This is especially helpful in environments where the nvidia driver is preconfigured on the nodes.

## Motivation

NVIDIA vGPUs fetch licenses using JWTs managed via the host's gridd service. In Kubernetes, manual JWT handling is complex. This initiative proposes a Kubernetes Operator to automate JWT distribution for vGPU nodes. A key function is to also monitor and report vGPU license status using Kubernetes-native mechanisms, simplifying management and providing integrated cluster visibility.

## Sequence Diagram

```mermaid
sequenceDiagram
    actor User
    participant K8sAPI as Kubernetes API
    participant VGPUController as VGPUToken Controller
    participant TokenDS as Token DaemonSet
    participant K8sNode as Kubernetes Node

    User->>K8sAPI: Create/Update VGPUToken CR
    K8sAPI->>VGPUController: Notify VGPUToken change
    VGPUController->>K8sAPI: Get VGPUToken details
    VGPUController->>K8sAPI: Get Token ConfigMap
    VGPUController->>K8sAPI: Create/Update Token DaemonSet
    K8sAPI->>TokenDS: Schedule Pods (nodeSelector match)

    loop For each matching node
        TokenDS->>K8sNode: Mount Token from ConfigMap
        Note over TokenDS,K8sNode: Kubelet syncs token to hostPath
        K8sNode->>TokenDS: Token deployment status
    end

    TokenDS->>VGPUController: Report Pod statuses
    VGPUController->>K8sAPI: Update VGPUToken Status (NodeTokenStatus)
```
