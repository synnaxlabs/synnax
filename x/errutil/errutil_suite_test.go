package errutil_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestErrutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Errutil Suite")
}
