package telem_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTelem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Telem Suite")
}