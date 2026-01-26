package color_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestColor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Color Suite")
}
