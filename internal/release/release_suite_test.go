package release

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestReleaseSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Release test suite")
}
