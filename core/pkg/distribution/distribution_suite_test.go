package distribution_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDistribution(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Distribution Suite")
}
