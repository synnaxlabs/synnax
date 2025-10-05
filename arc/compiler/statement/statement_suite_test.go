package statement_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var bCtx = context.Background()

func TestStatement(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Statement Compiler Suite")
}
