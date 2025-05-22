// Copyright 2025 Nutanix. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

/*
Copyright 2023 Nutanix

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

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/davecgh/go-spew/spew"
	nkpv1alpha1 "github.com/nutanix-cloud-native/vgpu-token-operator/api/v1alpha1"
	"github.com/nutanix-cloud-native/vgpu-token-operator/pkg/generator"
	"github.com/nutanix-cloud-native/vgpu-token-operator/test/e2e/framework/helm"
	"github.com/nutanix-cloud-native/vgpu-token-operator/test/e2e/framework/nutanix"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	. "sigs.k8s.io/cluster-api/test/e2e"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func generateRandomName(name string) string {
	randomSuffix := util.RandomString(6)
	return fmt.Sprintf("%s-%s", name, randomSuffix)
}

var _ = Describe("Nutanix Virtual GPU", Label("vgpu-token-operator"), func() {
	const specName = "vgpu"

	var (
		namespace        *corev1.Namespace
		clusterName      string
		clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult
		cancelWatches    context.CancelFunc
	)

	BeforeEach(func() {
		clusterName = generateRandomName(specName)
		clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)

		Expect(bootstrapClusterProxy).NotTo(BeNil(), "BootstrapClusterProxy can't be nil")

		Byf("Creating a namespace for hosting the %q test spec", specName)
		namespace, cancelWatches = framework.CreateNamespaceAndWatchEvents(ctx, framework.CreateNamespaceAndWatchEventsInput{
			Creator:   bootstrapClusterProxy.GetClient(),
			ClientSet: bootstrapClusterProxy.GetClientSet(),
			Name:      generateRandomName(specName),
			LogFolder: filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName()),
		})
		By(
			"Reserving an IP address for the workload cluster control plane endpoint",
		)
		nutanixClient, err := nutanix.NewV4Client(
			nutanix.CredentialsFromCAPIE2EConfig(e2eConfig),
		)
		Expect(err).ToNot(HaveOccurred(), "failed to create client")

		controlPlaneEndpointIP, unreserveControlPlaneEndpointIP, err := nutanix.ReserveIP(
			e2eConfig.MustGetVariable("NUTANIX_SUBNET_NAME"),
			e2eConfig.MustGetVariable(
				"NUTANIX_PRISM_ELEMENT_CLUSTER_NAME_CONTROL_PLANE",
			),
			nutanixClient,
		)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(unreserveControlPlaneEndpointIP)
		e2eConfig.Variables["CONTROL_PLANE_ENDPOINT_IP"] = controlPlaneEndpointIP
		By("Reinitializing clusterctl repository with new controlPlaneEndpointIP variable")
		clusterctlConfigPath = createClusterctlLocalRepository(e2eConfig, filepath.Join(artifactFolder, "repository"))
	})

	AfterEach(func() {
		Byf("Dumping logs from the %q workload cluster", clusterName)

		bootstrapClusterProxy.CollectWorkloadClusterLogs(ctx, namespace.Name, clusterName, filepath.Join(artifactFolder, "clusters", clusterName))

		Byf("Dumping all the Cluster API resources in the %s namespace", namespace)

		// Dump all Cluster API related resources to artifacts before deleting them.
		framework.DumpAllResources(ctx, framework.DumpAllResourcesInput{
			Lister:               bootstrapClusterProxy.GetClient(),
			KubeConfigPath:       bootstrapClusterProxy.GetKubeconfigPath(),
			ClusterctlConfigPath: clusterctlConfigPath,
			Namespace:            namespace.Name,
			LogPath:              filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName(), "resources"),
		})

		if !skipCleanup {
			Byf("Deleting cluster %s/%s", namespace, clusterName)
			// While https://github.com/kubernetes-sigs/cluster-api/issues/2955 is addressed in future iterations, there is a chance
			// that cluster variable is not set even if the cluster exists, so we are calling DeleteAllClustersAndWait
			// instead of DeleteClusterAndWait
			framework.DeleteAllClustersAndWait(ctx, framework.DeleteAllClustersAndWaitInput{
				ClusterProxy:         bootstrapClusterProxy,
				ClusterctlConfigPath: clusterctlConfigPath,
				Namespace:            namespace.Name,
			}, e2eConfig.GetIntervals("default", "wait-delete-cluster")...)

			framework.DeleteNamespace(
				ctx,
				framework.DeleteNamespaceInput{
					Deleter: bootstrapClusterProxy.GetClient(),
					Name:    namespace.Name,
				},
				e2eConfig.GetIntervals("default", "wait-delete-cluster")...)
		}
		cancelWatches()
	})

	It("Create a cluster with virtual GPUs", func() {
		const flavor = "vgpu"
		Expect(namespace).NotTo(BeNil())
		By("Creating a workload cluster", func() {
			cc := clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     clusterctlConfigPath,
				KubeconfigPath:           bootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
				Flavor:                   flavor,
				Namespace:                namespace.Name,
				ClusterName:              clusterName,
				KubernetesVersion:        e2eConfig.MustGetVariable(KubernetesVersion),
				ControlPlaneMachineCount: ptr.To(int64(1)),
				WorkerMachineCount:       ptr.To(int64(1)),
			}

			clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
				ClusterProxy:                 bootstrapClusterProxy,
				ConfigCluster:                cc,
				WaitForClusterIntervals:      e2eConfig.GetIntervals("", "wait-cluster"),
				WaitForControlPlaneIntervals: e2eConfig.GetIntervals("", "wait-control-plane"),
				WaitForMachineDeployments:    e2eConfig.GetIntervals("", "wait-worker-nodes"),
			}, clusterResources)

			By("Fetching the workload cluster")
			cluster := clusterResources.Cluster
			selfHostedClusterProxy := bootstrapClusterProxy.GetWorkloadCluster(
				ctx,
				cluster.Namespace,
				cluster.Name,
			)
			workloadClient := selfHostedClusterProxy.GetClient()

			tokenValue := os.Getenv("VGPU_TOKEN_SECRET")
			Expect(tokenValue).ToNot(BeEmpty())
			tokenCRName := "test-token"
			secretName := "client-config-token"
			vgpuSecretsData := []map[string]string{
				{
					"secretName": secretName,
					"value":      tokenValue,
				},
			}

			vgpuTokenObjectsData := []map[string]interface{}{
				{
					"tokenName":       tokenCRName,
					"tokenSecretName": secretName,
				},
			}

			combinedValues := map[string]interface{}{
				"vgpuTokens":       vgpuSecretsData,
				"vgpuTokenObjects": vgpuTokenObjectsData,
			}

			combinedValuesJSON, err := json.Marshal(combinedValues)
			Expect(err).NotTo(HaveOccurred())

			token := nkpv1alpha1.VGPUToken{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tokenCRName,
					Namespace: helm.Namespace,
				},
			}

			By("Deploying vgpu helm chart")
			err = helm.DeployVGPUChart(selfHostedClusterProxy.GetKubeconfigPath(), helmChartDir, vgpuImageOCIRepository, helmChartVersion, string(combinedValuesJSON))
			Expect(err).ToNot(HaveOccurred(), "expected to install helm chart")

			By("Checking token status")
			Eventually(func() []metav1.Condition {
				err = workloadClient.Get(ctx, ctrlclient.ObjectKeyFromObject(&token), &token)
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
						"Type":   Equal(nkpv1alpha1.ConditionPropagated),
						"Status": Equal(metav1.ConditionTrue),
					}),
				),
			)

			By("Checking observing daemonset status")
			ds := appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      generator.FormatDaemonSetName(token.GetName()),
					Namespace: helm.Namespace,
				},
			}
			Eventually(func() bool {
				err = workloadClient.Get(ctx, ctrlclient.ObjectKeyFromObject(&ds), &ds)
				if err != nil {
					if apierrors.IsNotFound(err) {
						return false
					}
					Expect(err).ShouldNot(HaveOccurred(), "unexpected error when getting daemonset.")

				}
				spew.Dump(ds.Status)
				return ds.Status.DesiredNumberScheduled > 0 && ds.Status.DesiredNumberScheduled == ds.Status.NumberAvailable
			}).WithTimeout(5*time.Minute).WithPolling(time.Second*5).Should(BeTrue(), "expected daemonset to have replicas")
		})
	})
})
