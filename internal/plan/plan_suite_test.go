package plan

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlanSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plan test suite")
}
