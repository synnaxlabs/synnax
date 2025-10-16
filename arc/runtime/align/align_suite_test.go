package align_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAlign(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Align Suite")
}
