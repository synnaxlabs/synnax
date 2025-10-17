package selector_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSelect(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Select Suite")
}