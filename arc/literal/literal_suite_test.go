package literal_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLiteral(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Literal Suite")
}
