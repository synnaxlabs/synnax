package confluence_test

import (
	"context"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/signal"
)

var _ = Describe("transform", func() {
	It("Should transform values correctly", func() {
		inlet := NewStream[int](3)
		outlet := NewStream[int](4)
		square := &LinearTransform[int, int]{}
		square.Transform = func(ctx context.Context, i int) (int, bool, error) {
			return i * i, true, nil
		}
		square.InFrom(inlet)
		square.OutTo(outlet)
		ctx, cancel := signal.WithCancel(context.Background())
		square.Flow(ctx)
		inlet.Inlet() <- 1
		inlet.Inlet() <- 2
		Expect(<-outlet.Outlet()).To(Equal(1))
		Expect(<-outlet.Outlet()).To(Equal(4))
		cancel()
		Expect(errors.Is(ctx.Wait(), context.Canceled)).To(BeTrue())
	})
})
