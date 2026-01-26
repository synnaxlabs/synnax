package ts_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TS Suite")
}
