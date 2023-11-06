package encoder_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEncoder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Encoder Suite")
}
