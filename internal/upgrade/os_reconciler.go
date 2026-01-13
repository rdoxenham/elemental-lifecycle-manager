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

type OSReconciler struct {
	planHandler
}

func NewOSReconciler(c client.Client) *OSReconciler {
	return &OSReconciler{
		planHandler: planHandler{Client: c},
	}
}

func (r *OSReconciler) ReconcilePlans(ctx context.Context, releaseName string, config *SUCPlanConfig) (*PhaseStatus, error) {
	logger := log.FromContext(ctx)

	if config == nil {
		return nil, fmt.Errorf("OS plan config is nil")
	}

	logger.Info("Reconciling OS upgrade SUC Plans",
		"image", config.Image,
		"version", config.Version,
		"release", releaseName)

	controlPlanePlan, err := r.getOrCreatePlan(ctx, plan.OSControlPlane(releaseName, config.Image, config.Version))
	if err != nil {
		return nil, fmt.Errorf("reconciling control plane plan: %w", err)
	}

	workerPlan, err := r.getOrCreatePlan(ctx, plan.OSWorker(releaseName, config.Image, config.Version))
	if err != nil {
		return nil, fmt.Errorf("reconciling worker plan: %w", err)
	}

	return r.aggregateStatus(controlPlanePlan, workerPlan), nil
}
