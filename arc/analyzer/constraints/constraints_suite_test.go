package constraints_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConstraints(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Constraints Suite")
}
