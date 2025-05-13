// Copyright 2025 Nutanix. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package helm

import (
	"fmt"
	"log"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/ptr"
)

const (
	namespace   = "vgpu-system"
	releaseName = "vgpu-token-operator"
)

func DeployVGPUChart(kubeconfig, chartDir, ociRepository, version string) error {
	settings := &genericclioptions.ConfigFlags{
		Namespace:  ptr.To(namespace),
		KubeConfig: ptr.To(kubeconfig),
	}

	actionConfig := &action.Configuration{}
	if err := actionConfig.Init(settings, namespace, "memory", log.Printf); err != nil {
		return fmt.Errorf("failed to initialize Helm action config: %w", err)
	}

	chart, err := loader.Load(chartDir)
	if err != nil {
		log.Fatalf("Failed to load chart: %v", err)
	}

	valOpts := &values.Options{
		StringValues: []string{
			fmt.Sprintf("controllerManager.container.image.repository=%s/vgpu-token-operator", ociRepository),
			fmt.Sprintf("controllerManager.container.image.tag=%s", version),
			fmt.Sprintf("vgpuCopy.image=%s/vgpu-token-copier:%s", ociRepository, version),
		},
	}

	envSettings := cli.New()
	if *settings.Namespace != "" {
		envSettings.SetNamespace(*settings.Namespace)
	}
	providers := getter.All(envSettings)
	vals, err := valOpts.MergeValues(providers)
	if err != nil {
		return fmt.Errorf("failed to merge values: %w", err)
	}

	installClient := action.NewInstall(actionConfig)
	installClient.Namespace = namespace
	installClient.CreateNamespace = true
	installClient.Wait = true
	installClient.ReleaseName = releaseName
	installClient.Timeout = time.Minute * 10
	_, err = installClient.Run(chart, vals)
	if err != nil {
		return fmt.Errorf("failed to install/upgrade: %w", err)
	}
	return nil
}
