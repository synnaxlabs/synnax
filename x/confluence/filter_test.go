package confluence_test

import (
	"context"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ApplySink", func() {
	It("Should filter values correctly", func() {
		ctx, cancel := signal.WithCancel(context.Background())
		inlet := confluence.NewStream[int](3)
		outlet := confluence.NewStream[int](3)
		filter := confluence.Filter[int]{
			Apply: func(ctx context.Context, x int) (bool, error) {
				return x%3 == 0, nil
			}}
		filter.InFrom(inlet)
		filter.OutTo(outlet)
		filter.Flow(ctx)
		inlet.Inlet() <- 1
		inlet.Inlet() <- 2
		inlet.Inlet() <- 3
		Expect(<-outlet.Outlet()).To(Equal(3))
		cancel()
		Expect(errors.Is(ctx.Wait(), context.Canceled)).To(BeTrue())
	})
})
