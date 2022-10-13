package seeker_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSeeker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Seeker Suite")
}
