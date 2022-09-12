package confluence_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

var _ = Describe("Linear", func() {
	It("Should transform values from inlet to outlet", func() {
		i := confluence.NewStream[int](1)
		o := confluence.NewStream[int](1)
		s := confluence.LinearTransform[int, int]{}
		s.Transform = func(ctx context.Context, i int) (int, bool, error) {
			return i * i, true, nil
		}
		s.InFrom(i)
		s.OutTo(o)
		ctx, cancel := signal.TODO()
		defer cancel()
		s.Flow(ctx)
		i.Inlet() <- 3
		Expect(<-o.Outlet()).To(Equal(9))
	})
	It("Should not send a value if the transform returns false", func() {
		i := confluence.NewStream[int](1)
		o := confluence.NewStream[int](1)
		s := confluence.LinearTransform[int, int]{}
		s.Transform = func(ctx context.Context, i int) (int, bool, error) {
			return 0, false, nil
		}
		s.InFrom(i)
		s.OutTo(o)
		ctx, cancel := signal.TODO()
		defer cancel()
		s.Flow(ctx, confluence.CloseInletsOnExit())
		i.Inlet() <- 3
		i.Close()
		_, ok := <-o.Outlet()
		Expect(ok).To(BeFalse())
	})
})
