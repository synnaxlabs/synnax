package override_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOverride(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Override Suite")
}
