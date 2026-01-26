package fmock_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFMock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FMock Suite")
}
