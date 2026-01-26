package pem_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPEM(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PEM Suite")
}
