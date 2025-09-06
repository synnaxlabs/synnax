package statement_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestStatement(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Statement Compiler Suite")
}
