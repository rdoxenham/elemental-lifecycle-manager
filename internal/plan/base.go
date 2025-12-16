package plan

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	upgradecattlev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
)

const (
	SUCNamespace = "cattle-system"
)

func basePlan(name string, drain bool) *upgradecattlev1.Plan {
	const (
		kind               = "Plan"
		apiVersion         = "upgrade.cattle.io/v1"
		serviceAccountName = "system-upgrade-controller"
	)

	plan := &upgradecattlev1.Plan{
		TypeMeta: metav1.TypeMeta{
			Kind:       kind,
			APIVersion: apiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: SUCNamespace,
		},
		Spec: upgradecattlev1.PlanSpec{
			ServiceAccountName: serviceAccountName,
		},
	}

	if drain {
		timeout := intstr.FromString("15m")
		deleteEmptyDirData := true
		ignoreDaemonSets := true
		plan.Spec.Drain = &upgradecattlev1.DrainSpec{
			Timeout:            &timeout,
			DeleteEmptydirData: &deleteEmptyDirData,
			IgnoreDaemonSets:   &ignoreDaemonSets,
			Force:              true,
		}
	}

	return plan
}
