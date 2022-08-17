package rand_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rand Suite")
}
