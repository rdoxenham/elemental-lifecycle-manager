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

package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		WithValidator(&ReleaseValidator{}).
		For(&Release{}).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-lifecycle-suse-com-v1alpha1-release,mutating=false,failurePolicy=fail,sideEffects=None,groups=lifecycle.suse.com,resources=releases,verbs=create;update;delete,versions=v1alpha1,name=vrelease.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &ReleaseValidator{}

type ReleaseValidator struct{}

func (r *ReleaseValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	release, ok := obj.(*Release)
	if !ok {
		return nil, fmt.Errorf("unexpected object type: %T", obj)
	}

	if release.Spec.Registry == "" {
		return nil, fmt.Errorf("release registry is required")
	}

	_, err := validateReleaseVersion(release.Spec.Version)
	return nil, err
}

func (r *ReleaseValidator) ValidateUpdate(_ context.Context, old, new runtime.Object) (admission.Warnings, error) {
	oldRelease, ok := old.(*Release)
	if !ok {
		return nil, fmt.Errorf("unexpected object type: %T", old)
	}

	newRelease, ok := new.(*Release)
	if !ok {
		return nil, fmt.Errorf("unexpected object type: %T", new)
	}

	newReleaseVersion, err := validateReleaseVersion(newRelease.Spec.Version)
	if err != nil {
		return nil, err
	}

	if newRelease.Spec.Registry == "" {
		return nil, fmt.Errorf("release registry is required")
	}

	if oldRelease.Status.Version != "" {
		indicator, err := newReleaseVersion.Compare(oldRelease.Status.Version)
		if err != nil {
			return nil, fmt.Errorf("comparing versions: %w", err)
		}

		switch indicator {
		case 0:
			return nil, fmt.Errorf("any edits over '%s' must come with an increment of the version", newRelease.Name)
		case -1:
			return nil, fmt.Errorf("new version must be greater than the currently applied one ('%s')", oldRelease.Status.Version)
		}
	}

	return nil, nil
}

func validateReleaseVersion(releaseVersion string) (*version.Version, error) {
	if releaseVersion == "" {
		return nil, fmt.Errorf("release version is required")
	}

	v, err := version.ParseSemantic(releaseVersion)
	if err != nil {
		return nil, fmt.Errorf("'%s' is not a semantic version", releaseVersion)
	}

	return v, nil
}

func (r *ReleaseValidator) ValidateDelete(context.Context, runtime.Object) (admission.Warnings, error) {
	return nil, fmt.Errorf("deleting release objects is not allowed")
}
