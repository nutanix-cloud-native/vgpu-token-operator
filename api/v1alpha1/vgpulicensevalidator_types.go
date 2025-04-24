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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VGPULicenseValidatorSpec defines a global license check object which uses LicenseVerificationCommand
// to verify if the node is correctly licensed or not.
type VGPULicenseValidatorSpec struct {
	// LicenseVerificationImage refers to the container image used to check if the
	// gridd service was correctly configured
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0"
	LicenseVerificationImage *string `json:"licenseVerificationImage,omitempty"`
	// NodeSelector refers to label selectors for finding the correct nodes to propagate the token to
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:={"nvidia.com/vgpu.present": "true"}
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// CheckIntervalDuration specifies how frequently to validate licenses.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="3m"
	CheckIntervalDuration *metav1.Duration `json:"checkIntervalDuration,omitempty"`

	// InitialDelayDuration waits after token propagation before first license check.
	// Helps avoid false negatives during GRID service initialization.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="2m"
	InitialDelayDuration *metav1.Duration `json:"initialDelayDuration,omitempty"`
}

// VGPULicenseValidatorStatus definess the state of the licenses on each node.
type VGPULicenseValidatorStatus struct {
	Conditions []metav1.Condition  `json:"conditions,omitempty"`
	Nodes      []NodeLicenseStatus `json:"nodes,omitempty"`
}

type NodeLicenseStatus struct {
	Name        string       `json:"name"`
	IsLicensed  bool         `json:"isLicensed"`
	LastChecked *metav1.Time `json:"lastChecked,omitempty"`
	Error       *string      `json:"error,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// VGPULicenseValidator is the Schema for the vgpulicensevalidators API.
type VGPULicenseValidator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VGPULicenseValidatorSpec   `json:"spec,omitempty"`
	Status VGPULicenseValidatorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VGPULicenseValidatorList contains a list of VGPULicenseValidator.
type VGPULicenseValidatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VGPULicenseValidator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VGPULicenseValidator{}, &VGPULicenseValidatorList{})
}
