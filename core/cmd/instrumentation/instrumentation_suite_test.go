package instrumentation_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInstrumentation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Instrumentation Suite")
}
