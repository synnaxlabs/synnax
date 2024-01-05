package rack_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRack(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rack Suite")
}
