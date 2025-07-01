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
	"helm.sh/helm/v3/pkg/strvals"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/ptr"
)

const (
	Namespace   = "vgpu-system"
	releaseName = "vgpu-token-operator"
)

func DeployVGPUChart(kubeconfig, chartDir, ociRepository, version string, extraValuesMap map[string]interface{}) error {
	settings := &genericclioptions.ConfigFlags{
		Namespace:  ptr.To(Namespace),
		KubeConfig: ptr.To(kubeconfig),
	}

	actionConfig := &action.Configuration{}
	if err := actionConfig.Init(settings, Namespace, "memory", log.Printf); err != nil {
		return fmt.Errorf("failed to initialize Helm action config: %w", err)
	}

	chart, err := loader.Load(chartDir)
	if err != nil {
		log.Fatalf("Failed to load chart: %v", err)
	}
	chart.Metadata.AppVersion = version
	chart.Metadata.Version = version

	valOpts := &values.Options{
		StringValues: []string{
			fmt.Sprintf("controllerManager.container.image.repository=%s/vgpu-token-operator", ociRepository),
			fmt.Sprintf("vgpuCopy.repository=%s/vgpu-token-copier", ociRepository),
			fmt.Sprintf("vgpuCopy.tag=%s-amd64", version),
		},
	}

	envSettings := cli.New()
	if *settings.Namespace != "" {
		envSettings.SetNamespace(*settings.Namespace)
	}
	providers := getter.All(envSettings)
	vals, err := valOpts.MergeValues(providers)
	for k, v := range extraValuesMap {
		// ParseJSON expects a string with format k1=v1
		if err := strvals.ParseJSON(fmt.Sprintf("%s=%s", k, v), vals); err != nil {
			return fmt.Errorf("failed parsing --set-json data %s=%s", k, v)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to merge values: %w", err)
	}

	installClient := action.NewInstall(actionConfig)
	installClient.Namespace = Namespace
	installClient.CreateNamespace = true
	installClient.Wait = true
	installClient.Atomic = true
	installClient.ReleaseName = releaseName
	installClient.Timeout = time.Minute * 10
	_, err = installClient.Run(chart, vals)
	if err != nil {
		return fmt.Errorf("failed to install/upgrade: %w", err)
	}
	return nil
}
