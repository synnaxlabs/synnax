package aspen_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAspen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Aspen Suite")
}
