package fnoop_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFNoOp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FNo-Op Suite")
}
