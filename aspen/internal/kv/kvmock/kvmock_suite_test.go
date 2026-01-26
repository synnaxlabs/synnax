package kvmock_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestKVMock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "KV Mock Suite")
}
