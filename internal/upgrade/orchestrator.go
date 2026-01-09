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

package upgrade

import (
	"context"
	"fmt"
	"strings"

	"github.com/suse/elemental/v3/pkg/manifest/api"
	"github.com/suse/elemental/v3/pkg/manifest/resolver"
)

// Orchestrator coordinates the upgrade process across all phases.
// It uses Rancher System Upgrade Controller Plans for OS and Kubernetes upgrades,
// and the Helm Controller for Helm chart deployments.
type Orchestrator struct {
	osPlanReconciler         SUCPlanReconciler
	kubernetesPlanReconciler SUCPlanReconciler
	helmChartReconciler      HelmChartReconciler
}

// NewOrchestrator creates a new upgrade orchestrator with the provided reconcilers.
func NewOrchestrator(osPlanReconciler, kubernetesPlanReconciler SUCPlanReconciler, helmChartReconciler HelmChartReconciler) *Orchestrator {
	return &Orchestrator{
		osPlanReconciler:         osPlanReconciler,
		kubernetesPlanReconciler: kubernetesPlanReconciler,
		helmChartReconciler:      helmChartReconciler,
	}
}

// BuildConfig creates a release upgrade specification from the resolved manifest.
// The upgrade is built by extracting configuration from the core platform
// and optionally merging with product extension components.
func (o *Orchestrator) BuildConfig(manifest *resolver.ResolvedManifest) (*Config, error) {
	if manifest == nil {
		return nil, fmt.Errorf("manifest is nil")
	}

	if manifest.CorePlatform == nil {
		return nil, fmt.Errorf("core platform manifest is required")
	}

	core := manifest.CorePlatform
	config := &Config{
		Version: core.Metadata.Version,
		OS: &SUCPlanConfig{
			Image:   core.Components.OperatingSystem.Image.Base,
			Version: core.Metadata.Version,
		},
	}

	kubernetesImage, kubernetesVersion := extractKubernetesImage(&core.Components.Systemd)
	if kubernetesImage == "" {
		return nil, fmt.Errorf("kubernetes image is required but not found in release manifest")
	}

	config.Kubernetes = &SUCPlanConfig{
		Image:   kubernetesImage,
		Version: kubernetesVersion,
	}

	if manifest.ProductExtension == nil {
		config.HelmCharts = o.buildHelmChartConfig(core.Components.Helm, nil)
	} else {
		product := manifest.ProductExtension
		config.HelmCharts = o.buildHelmChartConfig(core.Components.Helm, product.Components.Helm)
	}

	return config, nil
}

// buildHelmChartConfig merges Helm configurations from core and product manifests.
func (o *Orchestrator) buildHelmChartConfig(core *api.Helm, product *api.Helm) *HelmChartConfig {
	config := &HelmChartConfig{
		Charts:       make([]*api.HelmChart, 0),
		Repositories: make([]*api.HelmRepository, 0),
	}

	// Add core charts and repositories
	if core != nil {
		config.Charts = append(config.Charts, core.Charts...)
		config.Repositories = append(config.Repositories, core.Repositories...)
	}

	// Add product charts and repositories
	if product != nil {
		config.Charts = append(config.Charts, product.Charts...)
		config.Repositories = append(config.Repositories, product.Repositories...)
	}

	if len(config.Charts) == 0 && len(config.Repositories) == 0 {
		return nil
	}

	return config
}

// Reconcile ensures all upgrade resources are in the desired state.
// It reconciles resources in order:
// 1. OS upgrade SUC Plan
// 2. Kubernetes upgrade SUC Plan
// 3. Helm charts via Helm Controller
//
// Each phase checks if resources exist and creates/updates them as needed.
// The reconciliation stops at the first phase that fails.
func (o *Orchestrator) Reconcile(ctx context.Context, config *Config) error {
	if config == nil {
		return fmt.Errorf("upgrade config is nil")
	}

	if err := o.osPlanReconciler.ReconcilePlan(ctx, config.OS); err != nil {
		return fmt.Errorf("reconciling OS upgrade SUC Plan: %w", err)
	}

	if err := o.kubernetesPlanReconciler.ReconcilePlan(ctx, config.Kubernetes); err != nil {
		return fmt.Errorf("reconciling Kubernetes upgrade SUC Plan: %w", err)
	}

	if config.HelmCharts != nil {
		if err := o.helmChartReconciler.ReconcileHelmCharts(ctx, config.HelmCharts); err != nil {
			return fmt.Errorf("reconciling Helm charts: %w", err)
		}
	}

	return nil
}

func extractKubernetesImage(systemd *api.Systemd) (image string, version string) {
	if systemd == nil {
		return "", ""
	}

	// Extract version from the respective systemd extension
	for _, ext := range systemd.Extensions {
		if strings.Contains(ext.Name, "rke2") {
			image = ext.Image
			break
		}
	}

	res := strings.Split(image, ":")
	if len(res) != 2 {
		return "", ""
	}

	return res[0], res[1]
}
