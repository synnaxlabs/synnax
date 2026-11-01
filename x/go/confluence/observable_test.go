// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package confluence_test

import (
	"context"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Observable", func() {
	Describe("ObservableTransformPublisher", func() {
		It(
			"Should forward transformed values from an observable to the outlet",
			func() {
				obs := observe.New[int]()
				outlet := NewStream[int](10)
				pub := &ObservableTransformPublisher[int, int]{
					Observable: obs,
					Transform: func(_ context.Context, v int) (int, bool, error) {
						return v * 2, true, nil
					},
				}
				pub.OutTo(outlet)
				ctx, cancel := signal.Isolated()
				defer cancel()
				pub.Flow(ctx)

				obs.Notify(context.Background(), 5)
				Eventually(outlet.Outlet()).Should(Receive(Equal(10)))

				obs.Notify(context.Background(), 3)
				Eventually(outlet.Outlet()).Should(Receive(Equal(6)))

				cancel()
				Expect(ctx.Wait()).To(MatchError(context.Canceled))
			},
		)

		It("Should not forward values when transform returns false", func() {
			obs := observe.New[int]()
			outlet := NewStream[int](10)
			pub := &ObservableTransformPublisher[int, int]{
				Observable: obs,
				Transform: func(_ context.Context, v int) (int, bool, error) {
					return v, v > 0, nil
				},
			}
			pub.OutTo(outlet)
			ctx, cancel := signal.Isolated()
			defer cancel()
			pub.Flow(ctx)

			obs.Notify(context.Background(), -1)
			obs.Notify(context.Background(), 5)
			Eventually(outlet.Outlet()).Should(Receive(Equal(5)))

			cancel()
			Expect(ctx.Wait()).To(MatchError(context.Canceled))
		})

		It("Should disconnect the observer and close the outlet on shutdown", func() {
			obs := observe.New[int]()
			outlet := NewStream[int](10)
			pub := &ObservableTransformPublisher[int, int]{
				Observable: obs,
				Transform: func(_ context.Context, v int) (int, bool, error) {
					return v, true, nil
				},
			}
			pub.OutTo(outlet)
			ctx, cancel := signal.Isolated()
			pub.Flow(ctx, CloseOutputInletsOnExit())

			obs.Notify(context.Background(), 1)
			Eventually(outlet.Outlet()).Should(Receive(Equal(1)))

			cancel()
			Expect(ctx.Wait()).To(MatchError(context.Canceled))
			Eventually(outlet.Outlet()).Should(BeClosed())
		})

		It(
			"Should not deadlock when publisher shuts down with a full outlet buffer",
			func() {
				obsCtx, obsCancel := signal.Isolated()
				defer obsCancel()
				obs := observe.NewAsync[int](obsCtx, signal.WithRetryOnPanic(100))

				outlet := NewStream[int](1)
				pub := &ObservableTransformPublisher[int, int]{
					Observable: obs,
					Transform: func(_ context.Context, v int) (int, bool, error) {
						return v, true, nil
					},
				}
				pub.OutTo(outlet)

				pubCtx, pubCancel := signal.Isolated()
				pub.Flow(pubCtx, CloseOutputInletsOnExit())

				obs.Notify(context.Background(), 1)
				Eventually(outlet.Outlet()).Should(Receive(Equal(1)))

				obs.Notify(context.Background(), 2)
				time.Sleep(50 * time.Millisecond)
				obs.Notify(context.Background(), 3)

				pubCancel()
				Eventually(
					pubCtx.Wait,
					2*time.Second,
				).Should(MatchError(context.Canceled))
			},
		)

		It("Should not leak goroutines when used with an async observer", func() {
			sCtx, cancel := signal.Isolated()
			defer cancel()
			obs := observe.NewAsync[int](sCtx, signal.WithRetryOnPanic(100))

			outlet := NewStream[int](10)
			pub := &ObservableTransformPublisher[int, int]{
				Observable: obs,
				Transform: func(_ context.Context, v int) (int, bool, error) {
					return v, true, nil
				},
			}
			pub.OutTo(outlet)

			pubCtx, pubCancel := signal.Isolated()
			// Mid-test baseline: assert pub.Flow's goroutines are cleaned up by
			// pubCancel below, ignoring obs's goroutines under sCtx.
			ShouldNotLeakGoroutines()
			pub.Flow(pubCtx, CloseOutputInletsOnExit())

			obs.Notify(context.Background(), 42)
			Eventually(outlet.Outlet()).Should(Receive(Equal(42)))

			pubCancel()
			Expect(pubCtx.Wait()).To(MatchError(context.Canceled))
			Eventually(outlet.Outlet()).Should(BeClosed())

			Expect(sCtx.Wait()).To(Succeed())
			cancel()
		})
	})

	Describe("ObservableSubscriber", func() {
		It("Should notify observers when values are sent to the sink", func() {
			sub := NewObservableSubscriber[int]()
			inlet := NewStream[int](10)
			sub.InFrom(inlet)

			var received atomic.Int64
			sub.OnChange(func(_ context.Context, v int) {
				received.Store(int64(v))
			})

			ctx, cancel := signal.Isolated()
			defer cancel()
			sub.Flow(ctx)

			inlet.Inlet() <- 7
			Eventually(received.Load).Should(Equal(int64(7)))

			inlet.Inlet() <- 99
			Eventually(received.Load).Should(Equal(int64(99)))

			cancel()
			Expect(ctx.Wait()).To(MatchError(context.Canceled))
		})

		It("Should notify multiple observers", func() {
			sub := NewObservableSubscriber[int]()
			inlet := NewStream[int](10)
			sub.InFrom(inlet)

			var count atomic.Int64
			sub.OnChange(func(_ context.Context, _ int) { count.Add(1) })
			sub.OnChange(func(_ context.Context, _ int) { count.Add(1) })

			ctx, cancel := signal.Isolated()
			defer cancel()
			sub.Flow(ctx)

			inlet.Inlet() <- 1
			Eventually(count.Load).Should(Equal(int64(2)))

			cancel()
			Expect(ctx.Wait()).To(MatchError(context.Canceled))
		})
	})
})
