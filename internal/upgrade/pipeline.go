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

	lifecyclev1alpha1 "github.com/suse/elemental-lifecycle-manager/api/v1alpha1"
)

// Pipeline executes upgrade phases in sequence, stopping when a phase
// is not yet complete. This allows the controller to resume from where
// it left off on the next reconciliation.
type Pipeline struct {
	handlers []PhaseHandler
}

// NewPipeline creates a new pipeline with the given handlers.
// Handlers are executed in the order they are provided.
func NewPipeline(handlers ...PhaseHandler) *Pipeline {
	return &Pipeline{
		handlers: handlers,
	}
}

// Reconcile executes each phase handler in sequence.
// It stops and returns when:
// - A phase returns an error (wrapped in PhaseError)
// - A phase has not yet succeeded (allowing retry on next reconcile)
// - All phases complete successfully
func (p *Pipeline) Reconcile(ctx context.Context, config *Config) (*Result, error) {
	result := &Result{
		PhaseStates: make(map[Phase]*PhaseStatus),
	}

	if config == nil {
		return result, fmt.Errorf("upgrade config is nil")
	}

	for _, handler := range p.handlers {
		// Skip phases that don't apply to this config
		if !handler.ShouldReconcile(config) {
			continue
		}

		status, err := handler.Reconcile(ctx, config)
		if err != nil {
			return result, &PhaseError{Phase: handler.Phase(), Err: err}
		}
		result.PhaseStates[handler.Phase()] = status

		// Stop if phase not complete - will resume on next reconcile
		if status.State != lifecyclev1alpha1.UpgradeSucceeded {
			return result, nil
		}
	}

	return result, nil
}

// OSPhaseHandler handles the OS upgrade phase.
type OSPhaseHandler struct {
	reconciler SUCPlanReconciler
}

// NewOSPhaseHandler creates a new OS phase handler.
func NewOSPhaseHandler(reconciler SUCPlanReconciler) *OSPhaseHandler {
	return &OSPhaseHandler{reconciler: reconciler}
}

func (h *OSPhaseHandler) Phase() Phase {
	return PhaseOS
}

func (h *OSPhaseHandler) ShouldReconcile(config *Config) bool {
	return config.OS != nil
}

func (h *OSPhaseHandler) Reconcile(ctx context.Context, config *Config) (*PhaseStatus, error) {
	return h.reconciler.ReconcilePlans(ctx, config.ReleaseName, config.OS)
}

// KubernetesPhaseHandler handles the Kubernetes upgrade phase.
type KubernetesPhaseHandler struct {
	reconciler SUCPlanReconciler
}

// NewKubernetesPhaseHandler creates a new Kubernetes phase handler.
func NewKubernetesPhaseHandler(reconciler SUCPlanReconciler) *KubernetesPhaseHandler {
	return &KubernetesPhaseHandler{reconciler: reconciler}
}

func (h *KubernetesPhaseHandler) Phase() Phase {
	return PhaseKubernetes
}

func (h *KubernetesPhaseHandler) ShouldReconcile(config *Config) bool {
	return config.Kubernetes != nil
}

func (h *KubernetesPhaseHandler) Reconcile(ctx context.Context, config *Config) (*PhaseStatus, error) {
	return h.reconciler.ReconcilePlans(ctx, config.ReleaseName, config.Kubernetes)
}

// HelmChartsPhaseHandler handles the Helm charts upgrade phase.
type HelmChartsPhaseHandler struct {
	reconciler HelmChartReconciler
}

// NewHelmChartsPhaseHandler creates a new Helm charts phase handler.
func NewHelmChartsPhaseHandler(reconciler HelmChartReconciler) *HelmChartsPhaseHandler {
	return &HelmChartsPhaseHandler{reconciler: reconciler}
}

func (h *HelmChartsPhaseHandler) Phase() Phase {
	return PhaseHelmCharts
}

func (h *HelmChartsPhaseHandler) ShouldReconcile(config *Config) bool {
	return config.HelmCharts != nil
}

func (h *HelmChartsPhaseHandler) Reconcile(ctx context.Context, config *Config) (*PhaseStatus, error) {
	return h.reconciler.ReconcileHelmCharts(ctx, config.HelmCharts)
}
