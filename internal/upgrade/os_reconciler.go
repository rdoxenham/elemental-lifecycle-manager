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

// OSReconciler reconciles OS upgrades via SUC Plans and verifies node state.
type OSReconciler struct {
	planHandler
}

func NewOSReconciler(c client.Client) *OSReconciler {
	return &OSReconciler{
		planHandler: planHandler{Client: c},
	}
}

func (r *OSReconciler) Phase() Phase {
	return PhaseOS
}

func (r *OSReconciler) ShouldReconcile(config *Config) bool {
	return config.OS != nil
}

func (r *OSReconciler) Reconcile(ctx context.Context, config *Config) (*PhaseStatus, error) {
	logger := log.FromContext(ctx)
	osConfig := config.OS

	logger.Info("Reconciling OS upgrade",
		"image", osConfig.Image,
		"version", osConfig.Version,
		"release", config.ReleaseName)

	allNodes, err := r.listNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing nodes: %w", err)
	}

	p := plan.OSControlPlane(config.ReleaseName, osConfig.Image, osConfig.Version, osConfig.DrainOpts.ControlPlane)
	controlPlanePlan, err := r.getOrCreatePlan(ctx, p)
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

	if !allNodesUpgraded(cpNodes, osConfig.PrettyName) {
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

	p = plan.OSWorker(config.ReleaseName, osConfig.Image, osConfig.Version, osConfig.DrainOpts.Worker)
	workerPlan, err := r.getOrCreatePlan(ctx, p)
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

	if !allNodesUpgraded(workerNodes, osConfig.PrettyName) {
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
