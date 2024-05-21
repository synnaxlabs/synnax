package state_test

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	ctx  = context.Background()
	_b   *mock.Builder
	dist distribution.Distribution
)

var _ = BeforeSuite(func() {
	_b = mock.NewBuilder()
	dist = _b.New(ctx)
})

var _ = AfterSuite(func() {
	Expect(_b.Close()).To(Succeed())
	Expect(_b.Cleanup()).To(Succeed())
})

func TestState(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "State Suite")
}
