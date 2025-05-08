// Copyright 2025 Nutanix. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DefaultDaemonSetRoleName is the default name for the
	// ServiceAccount,Role,and RoleBinding used by the Token DaemonSet.
	DefaultDaemonSetRoleName = "vgpu-token-propagator"
)

func GenerateServiceAccount(namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultDaemonSetRoleName,
			Namespace: namespace,
		},
	}
}

// GenerateRole generates a Kubernetes Role object with necessary permissions for the token propagator.
func GenerateRole(namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultDaemonSetRoleName,
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""}, // "" indicates the core API group
				Resources: []string{"secrets"},
				Verbs:     []string{"get"},
			},
		},
	}
}

// GenerateRoleBinding generates a Kubernetes RoleBinding object that links the ServiceAccount and Role.
func GenerateRoleBinding(namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultDaemonSetRoleName,
			Namespace: namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     DefaultDaemonSetRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      DefaultDaemonSetRoleName,
				Namespace: namespace,
			},
		},
	}
}
