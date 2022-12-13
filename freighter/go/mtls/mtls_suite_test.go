package mtls_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMtls(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mtls Suite")
}
