package bounds_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBounds(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bounds Suite")
}
