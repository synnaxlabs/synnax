package allocate_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAlloc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Allocate Suite")
}
