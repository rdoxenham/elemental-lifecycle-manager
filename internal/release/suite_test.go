package release

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestActionSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Action test suite")
}
