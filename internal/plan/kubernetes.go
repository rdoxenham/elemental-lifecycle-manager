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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	kubernetesControlPlaneBaseName = "elemental-kubernetes-control-plane"
	kubernetesWorkerBaseName       = "elemental-kubernetes-worker"
)

// kubernetesControlPlaneName returns the full plan name for the given version.
func kubernetesControlPlaneName(version string) string {
	return fmt.Sprintf("%s-%s", kubernetesControlPlaneBaseName, SanitizeVersion(version))
}

// kubernetesWorkerName returns the full plan name for the given version.
func kubernetesWorkerName(version string) string {
	return fmt.Sprintf("%s-%s", kubernetesWorkerBaseName, SanitizeVersion(version))
}

// KubernetesControlPlane builds a SUC Plan for Kubernetes upgrades on control plane nodes.
func KubernetesControlPlane(releaseName, k8sImage, version string) *upgradecattlev1.Plan {
	p := basePlan(kubernetesControlPlaneName(version), true)
	p.Labels = map[string]string{
		ReleaseNameLabel:    releaseName,
		ReleaseVersionLabel: SanitizeVersion(version),
	}
	p.Spec.Version = version
	p.Spec.Concurrency = 1
	p.Spec.NodeSelector = &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      controlPlaneLabel,
				Operator: "In",
				Values:   []string{"true"},
			},
		},
	}
	p.Spec.Upgrade = &upgradecattlev1.ContainerSpec{
		Image: upgradeImage,
		// TODO: Fill in upgrade execution
		Command: []string{""},
		Args:    []string{""},
	}
	return p
}

// KubernetesWorker builds a SUC Plan for Kubernetes upgrades on worker nodes.
func KubernetesWorker(releaseName, k8sImage, version string) *upgradecattlev1.Plan {
	p := basePlan(kubernetesWorkerName(version), true)
	p.Labels = map[string]string{
		ReleaseNameLabel:    releaseName,
		ReleaseVersionLabel: SanitizeVersion(version),
	}
	p.Spec.Version = version
	p.Spec.Concurrency = 1
	p.Spec.NodeSelector = &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      controlPlaneLabel,
				Operator: "NotIn",
				Values:   []string{"true"},
			},
		},
	}
	p.Spec.Upgrade = &upgradecattlev1.ContainerSpec{
		// TODO: Fill in upgrade execution
		Command: []string{""},
		Args:    []string{""},
	}
	return p
}
