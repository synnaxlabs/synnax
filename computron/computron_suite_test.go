package computron_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestComputron(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Computron Suite")
}
