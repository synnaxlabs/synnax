package pledge_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestMembership(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Membership Suite")
}
