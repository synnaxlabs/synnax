package clamp_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestClamp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Clamp Suite")
}
