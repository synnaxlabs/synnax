package signal_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSignal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Conductor Suite")
}
