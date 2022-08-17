package alamos_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAlamos(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Alamos Suite")
}
