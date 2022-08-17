package atomic_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAtomic(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Atomic Suite")
}
