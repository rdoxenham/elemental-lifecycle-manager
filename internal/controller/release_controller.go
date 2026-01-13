/*
Copyright 2025-2026.

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

package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	lifecyclev1alpha1 "github.com/suse/elemental-lifecycle-manager/api/v1alpha1"
	"github.com/suse/elemental-lifecycle-manager/internal/upgrade"
	"github.com/suse/elemental/v3/pkg/manifest/resolver"
)

const requeueInterval = 30 * time.Second

// ReleaseReconciler reconciles a Release object
type ReleaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	RetrieveManifest func(ctx context.Context, registry, version string) (*resolver.ResolvedManifest, error)
	Orchestrator     *upgrade.Orchestrator
}

// +kubebuilder:rbac:groups=lifecycle.suse.com,resources=releases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=lifecycle.suse.com,resources=releases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=lifecycle.suse.com,resources=releases/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=nodes,verbs=watch;list
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=upgrade.cattle.io,resources=plans,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Release")

	release := &lifecyclev1alpha1.Release{}

	if err := r.Get(ctx, req.NamespacedName, release); err != nil {
		logger.Error(err, "unable to fetch Release")

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	result, err := r.reconcileNormal(ctx, release)

	// Attempt to update the release status before returning.
	return result, errors.Join(err, r.Status().Update(ctx, release))
}

func (r *ReleaseReconciler) reconcileNormal(ctx context.Context, release *lifecyclev1alpha1.Release) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Upgrade to the platform requested",
		"version", release.Spec.Version,
		"registry", release.Spec.Registry)

	initializePendingConditions(release)
	defer updateAppliedCondition(release)

	manifest, err := r.getOrRetrieveManifest(ctx, release)
	if err != nil {
		setCondition(release, lifecyclev1alpha1.ConditionManifestResolved, metav1.ConditionFalse,
			lifecyclev1alpha1.UpgradeFailed, fmt.Sprintf("Failed to retrieve release manifest: %v", err))
		return ctrl.Result{}, fmt.Errorf("retrieving release manifest: %w", err)
	}
	setCondition(release, lifecyclev1alpha1.ConditionManifestResolved, metav1.ConditionTrue,
		lifecyclev1alpha1.UpgradeSucceeded, "Release manifest retrieved successfully")

	config, err := r.Orchestrator.BuildConfig(manifest, release.Name)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("building upgrade config: %w", err)
	}

	logger.Info("Reconciling upgrade", "version", config.Version)

	result, err := r.Orchestrator.Reconcile(ctx, config)
	if err != nil {
		setPhaseConditionFromError(release, err)
		return ctrl.Result{}, fmt.Errorf("reconciling upgrade: %w", err)
	}

	updatePhaseConditions(release, result)

	if result.AllComplete() {
		release.Status.Version = config.Version
		return ctrl.Result{}, nil
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&lifecyclev1alpha1.Release{}).
		Complete(r)
}
