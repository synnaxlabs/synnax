package plumber_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlumber(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plumber Suite")
}
