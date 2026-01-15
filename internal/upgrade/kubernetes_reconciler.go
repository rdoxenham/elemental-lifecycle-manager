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

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	lifecyclev1alpha1 "github.com/suse/elemental-lifecycle-manager/api/v1alpha1"
	"github.com/suse/elemental-lifecycle-manager/internal/plan"
)

// KubernetesReconciler reconciles SUC Plans for Kubernetes upgrades.
type KubernetesReconciler struct {
	planHandler
}

// NewKubernetesReconciler creates a new Kubernetes reconciler.
func NewKubernetesReconciler(c client.Client) *KubernetesReconciler {
	return &KubernetesReconciler{
		planHandler: planHandler{Client: c},
	}
}

// ReconcilePlans ensures the SUC Plans for Kubernetes upgrades exist and returns their status.
func (r *KubernetesReconciler) ReconcilePlans(ctx context.Context, releaseName string, config *SUCPlanConfig) (*PhaseStatus, error) {
	logger := log.FromContext(ctx)

	if config == nil {
		return nil, fmt.Errorf("kubernetes plan config is nil")
	}

	logger.Info("Reconciling Kubernetes upgrade",
		"image", config.Image,
		"version", config.Version,
		"release", releaseName)

	allNodes, err := r.listNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing nodes: %w", err)
	}

	controlPlanePlan, err := r.getOrCreatePlan(ctx, plan.KubernetesControlPlane(releaseName, config.Image, config.Version))
	if err != nil {
		return nil, fmt.Errorf("reconciling control plane plan: %w", err)
	}

	if status := checkPlanFailure(controlPlanePlan); status != nil {
		return status, nil
	}

	cpNodes, err := filterNodesBySelector(allNodes, controlPlanePlan.Spec.NodeSelector)
	if err != nil {
		return nil, fmt.Errorf("filtering control plane nodes: %w", err)
	}

	if !allNodesAtKubernetesVersion(cpNodes, config.Version) {
		return &PhaseStatus{
			State:   lifecyclev1alpha1.UpgradeInProgress,
			Message: "Control plane nodes are being upgraded",
		}, nil
	}

	logger.Info("Control plane nodes upgraded", "count", len(cpNodes))

	if isControlPlaneOnlyCluster(allNodes) {
		logger.Info("Control-plane-only cluster detected, skipping worker upgrade")
		return &PhaseStatus{
			State:   lifecyclev1alpha1.UpgradeSucceeded,
			Message: "All cluster nodes are upgraded (control-plane-only cluster)",
		}, nil
	}

	workerPlan, err := r.getOrCreatePlan(ctx, plan.KubernetesWorker(releaseName, config.Image, config.Version))
	if err != nil {
		return nil, fmt.Errorf("reconciling worker plan: %w", err)
	}

	if status := checkPlanFailure(workerPlan); status != nil {
		return status, nil
	}

	workerNodes, err := filterNodesBySelector(allNodes, workerPlan.Spec.NodeSelector)
	if err != nil {
		return nil, fmt.Errorf("filtering worker nodes: %w", err)
	}

	if !allNodesAtKubernetesVersion(workerNodes, config.Version) {
		return &PhaseStatus{
			State:   lifecyclev1alpha1.UpgradeInProgress,
			Message: "Worker nodes are being upgraded",
		}, nil
	}

	logger.Info("All nodes upgraded", "controlPlane", len(cpNodes), "workers", len(workerNodes))

	return &PhaseStatus{
		State:   lifecyclev1alpha1.UpgradeSucceeded,
		Message: fmt.Sprintf("All %d nodes upgraded successfully", len(cpNodes)+len(workerNodes)),
	}, nil
}
