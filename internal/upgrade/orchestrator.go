/*
Copyright 2026.

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

	lifecyclev1alpha1 "github.com/suse/elemental-lifecycle-manager/api/v1alpha1"
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
func (o *Orchestrator) BuildConfig(manifest *resolver.ResolvedManifest, releaseName string) (*Config, error) {
	if manifest == nil {
		return nil, fmt.Errorf("manifest is nil")
	}

	if manifest.CorePlatform == nil {
		return nil, fmt.Errorf("core platform manifest is required")
	}

	core := manifest.CorePlatform
	config := &Config{
		ReleaseName: releaseName,
		Version:     core.Metadata.Version,
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

// Reconcile ensures all upgrade resources are in the desired state and returns their status.
// It reconciles resources in order:
// 1. OS upgrade SUC Plan
// 2. Kubernetes upgrade SUC Plan
// 3. Helm charts via Helm Controller
//
// Each phase checks if resources exist and creates them as needed.
// Returns a Result containing the status of each phase.
// On failure, returns a PhaseError with details about the failed phase.
func (o *Orchestrator) Reconcile(ctx context.Context, config *Config) (*Result, error) {
	result := &Result{
		PhaseStates: make(map[Phase]*PhaseStatus),
	}

	if config == nil {
		return result, fmt.Errorf("upgrade config is nil")
	}

	osState, err := o.osPlanReconciler.ReconcilePlans(ctx, config.ReleaseName, config.OS)
	if err != nil {
		return result, &PhaseError{Phase: PhaseOS, Err: err}
	}
	result.PhaseStates[PhaseOS] = osState

	k8sState, err := o.kubernetesPlanReconciler.ReconcilePlans(ctx, config.ReleaseName, config.Kubernetes)
	if err != nil {
		return result, &PhaseError{Phase: PhaseKubernetes, Err: err}
	}
	result.PhaseStates[PhaseKubernetes] = k8sState

	if config.HelmCharts != nil {
		if err := o.helmChartReconciler.ReconcileHelmCharts(ctx, config.HelmCharts); err != nil {
			return result, &PhaseError{Phase: PhaseHelmCharts, Err: err}
		}
		// TODO: HelmChartReconciler should return PhaseStatus
		result.PhaseStates[PhaseHelmCharts] = &PhaseStatus{State: lifecyclev1alpha1.UpgradeSucceeded, Message: "Helm charts reconciled"}
	}

	return result, nil
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
