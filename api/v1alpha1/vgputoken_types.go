// Copyright 2025 Nutanix. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VGPUTokenSpec defines how to find the token configmap and propagate it to nodes that use a set of labels.
type VGPUTokenSpec struct {
	// TokenConfigmap refers to the configmap that contains the token contents
	// this must be in the same namesapce as the token object
	// +kubebuilder:validation:Required
	TokenSecret corev1.LocalObjectReference `json:"tokenSecret"`

	// HostDirectoryPath refers to the path to mount the Token on for guest VMs.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="/etc/nvidia/ClientConfigToken/"
	HostDirectoryPath *string `json:"hostDirectoryPath,omitempty"`

	// NodeSelector refers to label selectors for finding the correct nodes to propagate the token to
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:={"nvidia.com/vgpu.present": "true", "nvidia.com/gpu.deploy.driver": "pre-installed"}
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

type VGPUTokenStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	Nodes      []NodeTokenStatus  `json:"nodes,omitempty"`
}

type NodeTokenStatus struct {
	Ready                    bool               `json:"ready"`
	Name                     string             `json:"name"`
	LastSync                 *metav1.Time       `json:"lastSync,omitempty"`
	Conditions               []metav1.Condition `json:"conditions,omitempty"`
	ObservedSecretGeneration string             `json:"observedSecretGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VGPUToken is the Schema for the vgputokens API.
type VGPUToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VGPUTokenSpec   `json:"spec,omitempty"`
	Status VGPUTokenStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// VGPUTokenList contains a list of VGPUToken.
type VGPUTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VGPUToken `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VGPUToken{}, &VGPUTokenList{})
}
