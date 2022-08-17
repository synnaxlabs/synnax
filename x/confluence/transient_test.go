package confluence_test

import (
	"context"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type transientSegment struct {
	confluence.TransientProvider
	confluence.LinearTransform[int, int]
}

func newTransientSegment() *transientSegment {
	t := &transientSegment{}
	t.ApplyTransform = func(ctx context.Context, i int) (int, bool, error) {
		if i == 3 {
			t.Transient() <- errors.New("error")
		}
		return i * i, true, nil
	}
	return t
}

var _ = Describe("TransientProvider", func() {
	Describe("TransientSource", func() {
		var (
			cancel context.CancelFunc
			trans  confluence.Stream[error]
			inlet  confluence.Stream[int]
			outlet confluence.Stream[int]
		)
		BeforeEach(func() {
			t := newTransientSegment()
			trans = confluence.NewStream[error](1)
			s := confluence.InjectTransient[int, int](trans, t)
			var ctx signal.Context
			ctx, cancel = signal.TODO()
			inlet = confluence.NewStream[int](0)
			s.InFrom(inlet)
			outlet = confluence.NewStream[int](0)
			s.OutTo(outlet)
			s.Flow(ctx, confluence.CloseInletsOnExit())
		})
		AfterEach(func() { cancel() })
		It("Should receive errors from the segment", func() {
			inlet.Inlet() <- 3
			Expect(errors.Is(<-trans.Outlet(), errors.New("error"))).To(BeTrue())
		})
		It("Should close the transient channel when the segments exit", func() {
			close(inlet.Inlet())
			_, ok := <-trans.Outlet()
			Expect(ok).To(BeFalse())
		})
	})
})
