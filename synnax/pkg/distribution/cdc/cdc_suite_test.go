package cdc_test

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCdc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cdc Suite")
}

var (
	_b   *mock.Builder
	dist distribution.Distribution
	ctx  = context.Background()
)

var _ = BeforeSuite(func() {
	_b = mock.NewBuilder()
	dist = _b.New(ctx)
})

var _ = AfterSuite(func() {
	Expect(_b.Close()).To(Succeed())
	Expect(_b.Cleanup()).To(Succeed())
})
