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

	"github.com/suse/elemental/v3/pkg/manifest/api"
)

// Phase represents a distinct phase in the upgrade process.
type Phase string

const (
	PhaseOS         Phase = "OperatingSystem"
	PhaseKubernetes Phase = "Kubernetes"
	PhaseHelm       Phase = "Helm"
)

// Status represents the current status of an upgrade phase.
type Status string

const (
	StatusPending    Status = "Pending"
	StatusInProgress Status = "InProgress"
	StatusCompleted  Status = "Completed"
	StatusFailed     Status = "Failed"
)

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
	// Version is the target release version.
	Version string
	// OS contains the SUC Plan configuration for OS upgrades.
	OS *SUCPlanConfig
	// Kubernetes contains the SUC Plan configuration for Kubernetes upgrades.
	Kubernetes *SUCPlanConfig
	// HelmCharts contains the Helm charts to deploy via Helm Controller.
	HelmCharts *HelmChartConfig
}

// SUCPlanReconciler defines the interface for reconciling a Rancher System Upgrade Controller Plan.
type SUCPlanReconciler interface {
	// ReconcilePlan ensures the SUC Plan exists and is up to date.
	ReconcilePlan(ctx context.Context, config *SUCPlanConfig) error
}

// HelmChartReconciler defines the interface for reconciling Helm Controller HelmChart resources.
type HelmChartReconciler interface {
	// ReconcileHelmCharts ensures the HelmChart resources exist and are up to date.
	ReconcileHelmCharts(ctx context.Context, config *HelmChartConfig) error
}
