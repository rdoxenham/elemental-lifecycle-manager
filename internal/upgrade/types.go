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

	lifecyclev1alpha1 "github.com/suse/elemental-lifecycle-manager/api/v1alpha1"
)

// Phase represents a distinct upgrade phase.
type Phase string

// Phase constants derived from condition types.
var (
	PhaseOS         = Phase(strings.TrimSuffix(lifecyclev1alpha1.ConditionOSUpgraded, "Upgraded"))
	PhaseKubernetes = Phase(strings.TrimSuffix(lifecyclev1alpha1.ConditionKubernetesUpgraded, "Upgraded"))
	PhaseHelmCharts = Phase(strings.TrimSuffix(lifecyclev1alpha1.ConditionHelmChartsUpgraded, "Upgraded"))
)

// AllPhases lists all upgrade phases in order.
var AllPhases = []Phase{
	PhaseOS,
	PhaseKubernetes,
	PhaseHelmCharts,
}

// ConditionType returns the condition type string for this phase.
func (p Phase) ConditionType() string {
	return string(p) + "Upgraded"
}

// PhaseStatus contains the status and details for an upgrade phase.
type PhaseStatus struct {
	State   string
	Message string
}

// PhaseError represents an error that occurred during a specific upgrade phase.
// The underlying Err should contain context-specific details from the reconciler.
type PhaseError struct {
	Phase Phase
	Err   error
}

func (e *PhaseError) Error() string {
	return fmt.Sprintf("%s upgrade failed: %v", e.Phase, e.Err)
}

// SUCPlanConfig contains configuration for creating a Rancher System Upgrade Controller Plan.
type SUCPlanConfig struct {
	// Image is the target image for the upgrade.
	Image string
	// Version is the target version.
	Version string
}

// HelmChartConfig contains configuration for Helm Controller HelmChart resources.
type HelmChartConfig struct {
	// Charts is the list of Helm charts to deploy/upgrade.
	Charts []*api.HelmChart
	// Repositories is the list of Helm repositories.
	Repositories []*api.HelmRepository
}

// Config represents a complete upgrade specification for all phases.
type Config struct {
	// ReleaseName is the name of the Release resource.
	ReleaseName string
	// Version is the target release version.
	Version string
	// OS contains the SUC Plan configuration for OS upgrades.
	OS *SUCPlanConfig
	// Kubernetes contains the SUC Plan configuration for Kubernetes upgrades.
	Kubernetes *SUCPlanConfig
	// HelmCharts contains the Helm charts to deploy via Helm Controller.
	HelmCharts *HelmChartConfig
}

// Result contains the outcome of the upgrade reconciliation.
type Result struct {
	// PhaseStates maps each phase to its current state.
	PhaseStates map[Phase]*PhaseStatus
}

// AllComplete returns true if all phases have succeeded.
func (r *Result) AllComplete() bool {
	for _, state := range r.PhaseStates {
		if state.State != lifecyclev1alpha1.UpgradeSucceeded {
			return false
		}
	}
	return true
}

// SUCPlanReconciler defines the interface for reconciling Rancher System Upgrade Controller Plans.
type SUCPlanReconciler interface {
	// ReconcilePlans ensures the SUC Plans for a particular configuration exist and returns their current status.
	ReconcilePlans(ctx context.Context, releaseName string, config *SUCPlanConfig) (*PhaseStatus, error)
}

// HelmChartReconciler defines the interface for reconciling Helm Controller HelmChart resources.
type HelmChartReconciler interface {
	// ReconcileHelmCharts ensures the HelmChart resources exist and are up to date.
	ReconcileHelmCharts(ctx context.Context, config *HelmChartConfig) error
}
