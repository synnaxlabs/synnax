package alignment_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAlignment(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Alignment Suite")
}
