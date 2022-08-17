package observe_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestObserve(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Observe Suite")
}
