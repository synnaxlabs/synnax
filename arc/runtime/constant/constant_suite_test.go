package constant_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConstant(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Constant Suite")
}
