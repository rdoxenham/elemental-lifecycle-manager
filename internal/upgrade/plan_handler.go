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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	upgradecattlev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	lifecyclev1alpha1 "github.com/suse/elemental-lifecycle-manager/api/v1alpha1"
)

type planHandler struct {
	client.Client
}

func (r *planHandler) getOrCreatePlan(ctx context.Context, desired *upgradecattlev1.Plan) (*upgradecattlev1.Plan, error) {
	logger := log.FromContext(ctx)

	existing := &upgradecattlev1.Plan{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      desired.Name,
		Namespace: desired.Namespace,
	}, existing)

	if apierrors.IsNotFound(err) {
		logger.Info("Creating SUC Plan", "name", desired.Name)
		if err = r.Create(ctx, desired); err != nil {
			return nil, err
		}
		return desired, nil
	}

	if err != nil {
		return nil, err
	}

	return existing, nil
}

func (r *planHandler) aggregateStatus(plans ...*upgradecattlev1.Plan) *PhaseStatus {
	var failures []string
	var inProgress bool

	for _, p := range plans {
		state, message := evaluatePlanStatus(p)
		switch state {
		case lifecyclev1alpha1.UpgradeFailed:
			failures = append(failures, fmt.Sprintf("[%s] %s", p.Name, message))
		case lifecyclev1alpha1.UpgradeInProgress:
			inProgress = true
		}
	}

	if len(failures) > 0 {
		return &PhaseStatus{
			State:   lifecyclev1alpha1.UpgradeFailed,
			Message: strings.Join(failures, "; "),
		}
	}

	if inProgress {
		return &PhaseStatus{
			State:   lifecyclev1alpha1.UpgradeInProgress,
			Message: "Upgrade plan in progress",
		}
	}

	return &PhaseStatus{
		State:   lifecyclev1alpha1.UpgradeSucceeded,
		Message: "All upgrade plans completed",
	}
}

func evaluatePlanStatus(p *upgradecattlev1.Plan) (string, string) {
	if len(p.Status.Applying) > 0 {
		return lifecyclev1alpha1.UpgradeInProgress, "Nodes being upgraded"
	}

	for _, cond := range p.Status.Conditions {
		if cond.Type == string(upgradecattlev1.PlanComplete) {
			if cond.Status == corev1.ConditionTrue {
				return lifecyclev1alpha1.UpgradeSucceeded, "Plan completed"
			}
			if cond.Status == corev1.ConditionFalse && cond.Reason != "" {
				return lifecyclev1alpha1.UpgradeFailed, cond.Message
			}
		}
	}

	return lifecyclev1alpha1.UpgradeInProgress, "Waiting for upgrade to complete"
}
