package plan

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/utils/ptr"
)

var _ = Describe("Base plan tests", func() {
	It("Properly creates a plan without draining options", func() {
		plan := basePlan("plan-1", false)

		Expect(plan).ToNot(BeNil())
		Expect(plan.TypeMeta.Kind).To(Equal("Plan"))
		Expect(plan.TypeMeta.APIVersion).To(Equal("upgrade.cattle.io/v1"))

		Expect(plan.ObjectMeta.Name).To(Equal("plan-1"))
		Expect(plan.ObjectMeta.Namespace).To(Equal("cattle-system"))
		Expect(plan.ObjectMeta.Labels).To(BeEmpty())
		Expect(plan.ObjectMeta.Annotations).To(BeEmpty())

		Expect(plan.Spec.ServiceAccountName).To(Equal("system-upgrade-controller"))
		Expect(plan.Spec.Drain).To(BeNil())
	})

	It("Properly creates a plan with draining options", func() {
		plan := basePlan("plan-1", true)

		Expect(plan).ToNot(BeNil())
		Expect(plan.TypeMeta.Kind).To(Equal("Plan"))
		Expect(plan.TypeMeta.APIVersion).To(Equal("upgrade.cattle.io/v1"))

		Expect(plan.ObjectMeta.Name).To(Equal("plan-1"))
		Expect(plan.ObjectMeta.Namespace).To(Equal("cattle-system"))
		Expect(plan.ObjectMeta.Labels).To(BeEmpty())
		Expect(plan.ObjectMeta.Annotations).To(BeEmpty())

		Expect(plan.Spec.ServiceAccountName).To(Equal("system-upgrade-controller"))

		Expect(plan.Spec.Drain).NotTo(BeNil())
		Expect(ptr.Deref(plan.Spec.Drain.DeleteEmptydirData, false)).To(BeTrue())
		Expect(ptr.Deref(plan.Spec.Drain.IgnoreDaemonSets, false)).To(BeTrue())
		Expect(plan.Spec.Drain.Force).To(BeTrue())
		Expect(plan.Spec.Drain.Timeout.String()).To(Equal("15m"))
	})
})
