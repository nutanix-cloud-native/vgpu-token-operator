// Copyright 2025 Nutanix. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	nkpv1alpha1 "github.com/nutanix-cloud-native/vgpu-token-operator/api/v1alpha1"
)

const (
	//nolint:gosec // these are not credentials
	HostTokenVolumeName = "host-token-dir"
	hostTokenMountPath  = "/host-token"

	SecretVolumeName = "vgpu-token"
	secretMountPath  = "/config"

	secretKey = "client_configuration_token.tok"
)

func GenerateDaemonSetForVGPUToken(
	token *nkpv1alpha1.VGPUToken,
	secret *corev1.Secret,
	image string,
) appsv1.DaemonSet {
	tolerations := []corev1.Toleration{
		{
			Key:      "nvidia.com/gpu",
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoSchedule,
		},
	}
	tokenPropagatorContainer := corev1.Container{
		Name:  "vgpu-token-propagater",
		Image: image,
		SecurityContext: &corev1.SecurityContext{
			RunAsUser: ptr.To(int64(0)),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
			ReadOnlyRootFilesystem: ptr.To(true),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      HostTokenVolumeName,
				MountPath: hostTokenMountPath,
			},
			{
				Name:      SecretVolumeName,
				MountPath: secretMountPath,
				ReadOnly:  true,
			},
		},
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: "File",
		ImagePullPolicy:          "IfNotPresent",
	}
	volumes := []corev1.Volume{
		{
			Name: HostTokenVolumeName,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: *token.Spec.HostDirectoryPath,
					Type: (*corev1.HostPathType)(ptr.To(string(corev1.HostPathDirectoryOrCreate))),
				},
			},
		},
		{
			Name: SecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secret.GetName(),
					Items: []corev1.KeyToPath{
						{
							Key:  secretKey,
							Path: secretKey,
						},
					},
				},
			},
		},
	}
	labels := map[string]string{
		"app.kubernetes.io/name": "vgpu-token-propagater",
	}
	ds := appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      FormatDaemonSetName(token.GetName()),
			Namespace: token.GetNamespace(),
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					PriorityClassName:            "system-cluster-critical",
					Tolerations:                  tolerations,
					ServiceAccountName:           DefaultDaemonSetRoleName,
					NodeSelector:                 token.Spec.NodeSelector,
					AutomountServiceAccountToken: ptr.To(false),
					Containers: []corev1.Container{
						tokenPropagatorContainer,
					},
					Volumes: volumes,
				},
			},
		},
	}
	return ds
}

func FormatDaemonSetName(tokenName string) string {
	suffix := "-token-daemonset"
	trimmedName := tokenName
	maxLength := 63 - len(suffix)
	if len(tokenName) > maxLength {
		trimmedName = tokenName[:maxLength]
	}
	return fmt.Sprintf("%s%s", trimmedName, suffix)
}
