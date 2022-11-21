package buffalo_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBuffalo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Buffalo Suite")
}
