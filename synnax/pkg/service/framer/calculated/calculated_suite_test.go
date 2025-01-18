package calculated_test

import (
	"context"
	"github.com/synnaxlabs/x/computronx"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	interpreter *computronx.Interpreter
	ctx         = context.Background()
)

var _ = BeforeSuite(func() {
	var err error
	interpreter, err = computronx.New()
	Expect(err).ToNot(HaveOccurred())
})

func TestCalculated(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Calculated Suite")
}
