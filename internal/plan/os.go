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

package plan

import (
	"fmt"

	upgradecattlev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	lifecyclev1alpha1 "github.com/suse/elemental-lifecycle-manager/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	osControlPlaneBaseName = "elemental-os-control-plane"
	osWorkerBaseName       = "elemental-os-worker"
)

// osControlPlaneName returns the full plan name for the given version.
func osControlPlaneName(version string) string {
	return fmt.Sprintf("%s-%s", osControlPlaneBaseName, lifecyclev1alpha1.SanitizeVersion(version))
}

// osWorkerName returns the full plan name for the given version.
func osWorkerName(version string) string {
	return fmt.Sprintf("%s-%s", osWorkerBaseName, lifecyclev1alpha1.SanitizeVersion(version))
}

// OSControlPlane builds a SUC Plan for OS upgrades on control plane nodes.
// Control plane nodes are upgraded first, without waiting for workers.
func OSControlPlane(releaseName, osImage, version string) *upgradecattlev1.Plan {
	p := basePlan(osControlPlaneName(version), true)
	p.Labels = map[string]string{
		lifecyclev1alpha1.ReleaseNameLabel:    releaseName,
		lifecyclev1alpha1.ReleaseVersionLabel: lifecyclev1alpha1.SanitizeVersion(version),
	}
	p.Spec.Version = version
	p.Spec.Concurrency = 1
	p.Spec.NodeSelector = &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      controlPlaneLabel,
				Operator: "In",
				Values: []string{
					"true",
				},
			},
		},
	}
	p.Spec.Upgrade = &upgradecattlev1.ContainerSpec{
		Image:   upgradeImage,
		Command: []string{"elemental3ctl"},
		Args:    []string{"upgrade", "--os-image", osImage},
	}
	return p
}

// OSWorker builds a SUC Plan for OS upgrades on worker nodes.
// Worker nodes wait for control plane upgrades to complete before starting.
func OSWorker(releaseName, osImage, version string) *upgradecattlev1.Plan {
	p := basePlan(osWorkerName(version), true)
	p.Labels = map[string]string{
		lifecyclev1alpha1.ReleaseNameLabel:    releaseName,
		lifecyclev1alpha1.ReleaseVersionLabel: lifecyclev1alpha1.SanitizeVersion(version),
	}
	p.Spec.Version = version
	p.Spec.Concurrency = 1
	p.Spec.NodeSelector = &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      controlPlaneLabel,
				Operator: "NotIn",
				Values: []string{
					"true",
				},
			},
		},
	}
	p.Spec.Upgrade = &upgradecattlev1.ContainerSpec{
		Image:   upgradeImage,
		Command: []string{"elemental3ctl"},
		Args:    []string{"upgrade", "--os-image", osImage},
	}
	return p
}
