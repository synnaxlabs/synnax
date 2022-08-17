package confluence_test

import (
	"context"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/signal"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"runtime"
	"time"
)

var _ = Describe("Emitter", func() {
	It("Should emit values at regular intervals", func() {
		e := &confluence.Emitter[int]{}
		e.Interval = 1 * time.Millisecond
		e.Emit = func(ctx context.Context) (int, error) {
			return 1, nil
		}
		ctx, cancel := signal.WithTimeout(context.TODO(), 100*time.Millisecond)
		defer cancel()
		stream := confluence.NewStream[int](0)
		e.OutTo(stream)
		e.Flow(ctx, confluence.CloseInletsOnExit())
		var received []int
		for v := range stream.Outlet() {
			received = append(received, v)
			runtime.Gosched()
		}
		Expect(len(received)).To(BeNumerically(">", 0))
	})

})
