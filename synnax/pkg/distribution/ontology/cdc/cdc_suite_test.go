package cdc_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCdc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cdc Suite")
}
