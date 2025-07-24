package fhttp_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFhttp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fhttp Suite")
}
