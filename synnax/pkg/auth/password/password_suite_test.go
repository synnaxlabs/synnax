package password_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPassword(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Password Suite")
}
