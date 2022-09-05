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
	t.Transform = func(ctx context.Context, i int) (int, bool, error) {
		if i == 3 {
			t.Transient() <- errors.New("error")
		}
		return i * i, true, nil
	}
	return t
}

var _ = Describe("TransientProvider", func() {
	var (
		ctx    signal.Context
		cancel context.CancelFunc
		trans  confluence.Stream[error]
		inlet  confluence.Stream[int]
		outlet confluence.Stream[int]
		t      *transientSegment
	)
	BeforeEach(func() {
		t = newTransientSegment()
		trans = confluence.NewStream[error](1)
		trans.SetInletAddress("transient")
		ctx, cancel = signal.TODO()
		inlet = confluence.NewStream[int](0)
		t.InFrom(inlet)
		outlet = confluence.NewStream[int](0)
		t.OutTo(outlet)
	})

	AfterEach(func() { cancel() })

	runSpecs := func() {
		It("Should receive errors from the segment", func() {
			inlet.Inlet() <- 3
			Expect(errors.Is(<-trans.Outlet(), errors.New("error"))).To(BeTrue())
		})
		It("Should close the transient channel when the segments exit", func() {
			inlet.Close()
			_, ok := <-trans.Outlet()
			Expect(ok).To(BeFalse())
		})
	}

	Describe("Transient", func() {
		BeforeEach(func() {
			s := confluence.InjectTransient[int, int](trans, t)
			s.Flow(ctx, confluence.CloseInletsOnExit())
		})
		runSpecs()
	})

	Describe("TransientSource", func() {
		BeforeEach(func() {
			s := confluence.InjectTransientSource[int](trans, t)
			s.Flow(ctx, confluence.CloseInletsOnExit())
		})
		runSpecs()
	})

	Describe("TransientSink", func() {
		BeforeEach(func() {
			s := confluence.InjectTransientSink[int](trans, t)
			s.Flow(ctx, confluence.CloseInletsOnExit(), confluence.CancelOnExitErr())
		})
		runSpecs()
	})
})
