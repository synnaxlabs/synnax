package stable_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestStable(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stable Suite")
}