// Copyright 2025 Nutanix. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	nkpv1alpha1 "github.com/nutanix-cloud-native/vgpu-token-operator/api/v1alpha1"
	"github.com/nutanix-cloud-native/vgpu-token-operator/pkg/generator"
)

func createTokenWithPrereqs(
	ctx context.Context,
	testNamespace string,
) *nkpv1alpha1.VGPUToken {
	name := "has-secret"
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
	}
	err := TestEnv.CreateObj(ctx, &secret)
	Expect(err).NotTo(HaveOccurred())
	token := nkpv1alpha1.VGPUToken{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: nkpv1alpha1.VGPUTokenSpec{
			TokenSecretRef: corev1.LocalObjectReference{
				Name: name,
			},
		},
	}
	err = TestEnv.CreateObj(ctx, &token)
	Expect(err).NotTo(HaveOccurred())
	Eventually(func() []metav1.Condition {
		err = TestEnv.Client.Get(ctx, ctrlclient.ObjectKeyFromObject(&token), &token)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			Expect(err).ShouldNot(HaveOccurred(), "unexpected error when getting token.")
		}
		return token.Status.Conditions
	}).Should(
		ContainElement(
			gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"Type":   Equal(nkpv1alpha1.ConditionTokenSecret),
				"Status": Equal(metav1.ConditionTrue),
			}),
		),
	)
	return &token
}

var _ = Describe("VGPUToken Controller", func() {
	var (
		ctx           context.Context
		testNamespace string
		ns            *corev1.Namespace
		err           error
		//nolint:gosec // not using credentials
		vGPUTokenPropagatorImage = "ghcr.io/nutanix-cloud-native/vgpu-token-copier:v0.0.0-dev.0"
	)
	BeforeEach(func() {
		ctx = context.Background()
		ns, err = TestEnv.CreateNamespace(ctx, "test")
		Expect(err).ToNot(HaveOccurred(), "failed to create test namespace")
		testNamespace = ns.Name
		err = (&VGPUTokenReconciler{
			Client:                   TestEnv.GetClient(),
			Scheme:                   TestEnv.GetScheme(),
			VGPUTokenPropagatorImage: vGPUTokenPropagatorImage,
		}).SetupWithManager(ctx, TestEnv.Manager)
	})
	AfterEach(func() {
		ctx = context.Background()
		err = TestEnv.Cleanup(ctx, ns)
		Expect(err).ToNot(HaveOccurred(), "failed to create test namespace")
	})
	It("should fail if the secret is not present", func() {
		name := "no-secret"
		token := nkpv1alpha1.VGPUToken{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: testNamespace,
			},
			Spec: nkpv1alpha1.VGPUTokenSpec{
				TokenSecretRef: corev1.LocalObjectReference{
					Name: "not-here",
				},
			},
		}
		err := TestEnv.CreateObj(ctx, &token)
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() []metav1.Condition {
			err = TestEnv.Client.Get(ctx, ctrlclient.ObjectKeyFromObject(&token), &token)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}
				Expect(err).ShouldNot(HaveOccurred(), "unexpected error when getting token.")

			}
			return token.Status.Conditions
		}).Should(
			ContainElement(
				gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Type":   Equal(nkpv1alpha1.ConditionTokenSecret),
					"Reason": Equal(nkpv1alpha1.ReasonTokenSecretNotFound),
				}),
			),
		)
	})
	It("should pass if secret is present", func() {
		name := "has-secret"
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: testNamespace,
			},
		}
		err := TestEnv.CreateObj(ctx, &secret)
		Expect(err).NotTo(HaveOccurred())
		token := nkpv1alpha1.VGPUToken{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: testNamespace,
			},
			Spec: nkpv1alpha1.VGPUTokenSpec{
				TokenSecretRef: corev1.LocalObjectReference{
					Name: name,
				},
			},
		}
		err = TestEnv.CreateObj(ctx, &token)
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() []metav1.Condition {
			err = TestEnv.Client.Get(ctx, ctrlclient.ObjectKeyFromObject(&token), &token)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}
				Expect(err).ShouldNot(HaveOccurred(), "unexpected error when getting token.")

			}
			return token.Status.Conditions
		}).Should(
			ContainElement(
				gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Type":   Equal(nkpv1alpha1.ConditionTokenSecret),
					"Status": Equal(metav1.ConditionTrue),
				}),
			),
		)
		ds := appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      generator.FormatDaemonSetName(token.GetName()),
				Namespace: testNamespace,
			},
		}
		Eventually(func() *appsv1.DaemonSet {
			err = TestEnv.Client.Get(ctx, ctrlclient.ObjectKeyFromObject(&ds), &ds)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}
				Expect(err).ShouldNot(HaveOccurred(), "unexpected error when getting token.")

			}
			return &ds
		}).Should(
			WithTransform(func(ds *appsv1.DaemonSet) corev1.PodSpec {
				return ds.Spec.Template.Spec
			}, &gstruct.FieldsMatcher{
				Fields: gstruct.Fields{
					"Containers": ContainElement(&gstruct.FieldsMatcher{
						Fields: gstruct.Fields{
							"Name":  Equal("vgpu-token-propagater"),
							"Image": Equal(vGPUTokenPropagatorImage),
							"VolumeMounts": And(
								HaveLen(2),
								ContainElement(&gstruct.FieldsMatcher{
									Fields: gstruct.Fields{
										"Name":      Equal(generator.HostTokenVolumeName),
										"MountPath": Equal("/host-token"),
									},
									IgnoreExtras:  true,
									IgnoreMissing: true,
								}),
								ContainElement(&gstruct.FieldsMatcher{
									Fields: gstruct.Fields{
										"Name":      Equal(generator.SecretVolumeName),
										"MountPath": Equal("/config"),
									},
									IgnoreExtras:  true,
									IgnoreMissing: true,
								}),
							),
						},
						IgnoreExtras:  true,
						IgnoreMissing: false,
					}),
					"NodeSelector": gstruct.MatchKeys(0, gstruct.Keys{
						"nvidia.com/vgpu.present":      Equal("true"),
						"nvidia.com/gpu.deploy.driver": Equal("pre-installed"),
					}),
					"Volumes": And(
						HaveLen(2),
						ContainElement(&gstruct.FieldsMatcher{
							Fields: gstruct.Fields{
								"Name": Equal(generator.HostTokenVolumeName),
								"HostPath": &gstruct.FieldsMatcher{
									Fields: gstruct.Fields{
										"Path": Equal(token.Spec.HostDirectoryPath),
									},
									IgnoreExtras:  true,
									IgnoreMissing: true,
								},
							},
							IgnoreExtras:  true,
							IgnoreMissing: true,
						}),
						ContainElement(&gstruct.FieldsMatcher{
							Fields: gstruct.Fields{
								"Name": Equal(generator.SecretVolumeName),
								"Secret": &gstruct.FieldsMatcher{
									IgnoreMissing: true,
									IgnoreExtras:  true,
									Fields: gstruct.Fields{
										"SecretName": Equal(token.Spec.TokenSecretRef.Name),
									},
								},
							},
							IgnoreExtras:  true,
							IgnoreMissing: true,
						}),
					),
				},
				IgnoreExtras:  true,
				IgnoreMissing: true,
			}),
			"DaemonSet should have expected pod spec",
		)
		role := rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      generator.DefaultDaemonSetRoleName,
				Namespace: testNamespace,
			},
		}
		Eventually(func() *rbacv1.Role {
			err = TestEnv.Client.Get(ctx, ctrlclient.ObjectKeyFromObject(&role), &role)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}
				Expect(err).ShouldNot(HaveOccurred(), "unexpected error when getting role.")

			}
			return &role
		}).ShouldNot(BeNil())
		roleBinding := rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      generator.DefaultDaemonSetRoleName,
				Namespace: testNamespace,
			},
		}
		Eventually(func() *rbacv1.RoleBinding {
			err = TestEnv.Client.Get(ctx, ctrlclient.ObjectKeyFromObject(&roleBinding), &roleBinding)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}
				Expect(err).ShouldNot(HaveOccurred(), "unexpected error when getting roleBinding.")

			}
			return &roleBinding
		}).ShouldNot(BeNil())
		sa := corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      generator.DefaultDaemonSetRoleName,
				Namespace: testNamespace,
			},
		}
		Eventually(func() *corev1.ServiceAccount {
			err = TestEnv.Client.Get(ctx, ctrlclient.ObjectKeyFromObject(&sa), &sa)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}
				Expect(err).ShouldNot(HaveOccurred(), "unexpected error when getting service account.")

			}
			return &sa
		}).ShouldNot(BeNil())
	})
	It("should create a new daemonset if one is deleted", func() {
		ctx := context.Background()
		token := createTokenWithPrereqs(
			ctx,
			testNamespace,
		)
		ds := appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      generator.FormatDaemonSetName(token.GetName()),
				Namespace: testNamespace,
			},
		}
		Eventually(func() *appsv1.DaemonSet {
			err = TestEnv.Client.Get(ctx, ctrlclient.ObjectKeyFromObject(&ds), &ds)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}
				Expect(err).ShouldNot(HaveOccurred(), "unexpected error when getting token.")

			}
			return &ds
		}).ShouldNot(BeNil(), "expected to create a daemonest")
		By("deleting the daemonest")
		err := TestEnv.Cleanup(
			ctx,
			&ds,
		)
		Expect(err).To(BeNil())

		err = TestEnv.Client.Get(ctx, ctrlclient.ObjectKeyFromObject(&ds), &ds)
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "expected not found error")

		Eventually(func() *appsv1.DaemonSet {
			err = TestEnv.Client.Get(ctx, ctrlclient.ObjectKeyFromObject(&ds), &ds)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}
				Expect(err).ShouldNot(HaveOccurred(), "unexpected error when getting token.")

			}
			return &ds
		}).ShouldNot(BeNil(), "expected daemonest to be recreated")
	})
})
