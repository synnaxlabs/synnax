package bit_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bit Suite")
}
