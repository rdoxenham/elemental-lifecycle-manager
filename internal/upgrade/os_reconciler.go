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

	upgradecattlev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
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

	controlPlaneTemplate := plan.OSControlPlane(config.ReleaseName, osConfig.Image, osConfig.Version, osConfig.DrainOpts.ControlPlane)
	workerTemplate := plan.OSWorker(config.ReleaseName, osConfig.Image, osConfig.Version, osConfig.DrainOpts.Worker)

	status, err := r.reconcileOSPlan(ctx, controlPlaneTemplate)
	if err != nil || status.State != lifecyclev1alpha1.PlanComplete {
		return status, err
	}

	status, err = r.reconcileOSPlan(ctx, workerTemplate)
	if err != nil || status.State != lifecyclev1alpha1.PlanComplete {
		return status, err
	}

	return &PhaseStatus{
		State:   lifecyclev1alpha1.UpgradeSucceeded,
		Message: "All nodes upgraded successfully",
	}, nil
}

func (r *OSReconciler) reconcileOSPlan(ctx context.Context, planTemplate *upgradecattlev1.Plan) (*PhaseStatus, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling OS upgrade plan",
		"plan", planTemplate.Name,
		"namespace", planTemplate.Namespace,
	)

	nodes, err := r.listNodesForPlan(ctx, planTemplate)
	if err != nil {
		return nil, fmt.Errorf("listing nodes for plan %s: %w", planTemplate.Name, err)
	}

	bootIDs := make(map[string]string, len(nodes.Items))
	for _, n := range nodes.Items {
		bootIDs[n.Name] = n.Status.NodeInfo.BootID
	}

	status, err := r.reconcilePlan(ctx, planTemplate)
	if err != nil {
		return nil, fmt.Errorf("reconciling plan %s: %w", planTemplate.Name, err)
	}

	if status.State != lifecyclev1alpha1.PlanComplete {
		return status, nil
	}

	upgradedNodes, err := r.listNodesForPlan(ctx, planTemplate)
	if err != nil {
		return nil, fmt.Errorf("listing upgraded nodes for plan %s: %w", planTemplate.Name, err)
	}

	// Safeguard for corner cases where the plan completes, but nodes are not yet ready.
	if !allNodesReady(upgradedNodes.Items) || !allNodesRebooted(upgradedNodes.Items, bootIDs) {
		return &PhaseStatus{
			State:   lifecyclev1alpha1.UpgradeInProgress,
			Message: fmt.Sprintf("Plan %s completed, waiting for node upgrade verification", planTemplate.Name),
		}, nil
	}

	nodeNames := make([]string, 0, len(upgradedNodes.Items))
	for _, n := range upgradedNodes.Items {
		nodeNames = append(nodeNames, n.Name)
	}

	logger.Info("OS upgrade plan completed",
		"plan", planTemplate.Name,
		"namespace", planTemplate.Namespace,
		"upgradedNodes", nodeNames,
	)
	return status, nil
}
