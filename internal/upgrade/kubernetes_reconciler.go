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

	logger.Info("Reconciling Kubernetes upgrade SUC Plans",
		"image", config.Image,
		"version", config.Version,
		"release", releaseName)

	controlPlanePlan, err := r.getOrCreatePlan(ctx, plan.KubernetesControlPlane(releaseName, config.Image, config.Version))
	if err != nil {
		return nil, fmt.Errorf("reconciling control plane plan: %w", err)
	}

	workerPlan, err := r.getOrCreatePlan(ctx, plan.KubernetesWorker(releaseName, config.Image, config.Version))
	if err != nil {
		return nil, fmt.Errorf("reconciling worker plan: %w", err)
	}

	return r.aggregateStatus(controlPlanePlan, workerPlan), nil
}
