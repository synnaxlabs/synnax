package math_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMath(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Math Suite")
}
