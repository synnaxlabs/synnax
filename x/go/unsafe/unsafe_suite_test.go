package unsafe_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUnsafe(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Unsafe Suite")
}
