package falamos_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFalamos(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Falamos Suite")
}
