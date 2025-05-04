package stringer_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestStringer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stringer Suite")
}
