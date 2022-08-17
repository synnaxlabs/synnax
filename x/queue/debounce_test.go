package queue_test

import (
	"context"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/queue"
	"github.com/arya-analytics/x/signal"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Debounce", func() {
	var (
		req    confluence.Stream[[]int]
		res    confluence.Stream[[]int]
		d      *queue.Debounce[int]
		ctx    signal.Context
		cancel context.CancelFunc
	)
	BeforeEach(func() {
		d = &queue.Debounce[int]{
			Config: queue.DebounceConfig{
				Interval:  30 * time.Millisecond,
				Threshold: 15,
			},
		}
		req = confluence.NewStream[[]int](10)
		res = confluence.NewStream[[]int](10)
		ctx, cancel = signal.TODO()
		d.InFrom(req)
		d.OutTo(res)
		d.Flow(ctx, confluence.CloseInletsOnExit())
	})
	AfterEach(func() { cancel() })
	It("Should flush the queue at a specified interval", func() {
		req.Inlet() <- []int{1, 2, 3, 4, 5}
		req.Inlet() <- []int{6, 7, 8, 9, 10}
		time.Sleep(50 * time.Millisecond)
		req.Close()
		Expect(ctx.Wait()).To(Succeed())
		responses := <-res.Outlet()
		Expect(responses).To(Equal([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))
	})
	It("Should flush teh queue when the threshold is reached", func() {
		req.Inlet() <- []int{1, 2, 3, 4, 5}
		req.Inlet() <- []int{6, 7, 8, 9, 10}
		req.Inlet() <- []int{11, 12, 13, 14, 15}
		req.Close()
		Expect(ctx.Wait()).To(Succeed())
		responses := <-res.Outlet()
		Expect(responses).To(Equal([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}))
	})
})
