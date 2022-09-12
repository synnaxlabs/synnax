package mock_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mock Suite")
}
