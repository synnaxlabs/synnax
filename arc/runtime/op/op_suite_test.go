package op_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Op Suite")
}
