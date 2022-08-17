package lock_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Locker Suite")
}
