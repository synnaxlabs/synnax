package testutil_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTestUtil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Utilities Suite")
}
