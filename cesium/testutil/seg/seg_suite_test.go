package seg_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSeg(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Seg Suite")
}
