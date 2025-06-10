package pebblekv_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPebblekv(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pebblekv Suite")
}
