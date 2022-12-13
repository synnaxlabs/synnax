package iter_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Iter Suite")
}
