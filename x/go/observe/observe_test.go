// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package observe_test

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
)

var _ = Describe("Observer", func() {
	It("Should notify all handlers", func(ctx SpecContext) {
		obs := observe.New[int]()
		var results []int
		obs.OnChange(func(ctx context.Context, v int) {
			results = append(results, v)
		})
		obs.OnChange(func(ctx context.Context, v int) {
			results = append(results, v*2)
		})
		obs.Notify(ctx, 5)
		Expect(results).To(ContainElements(5, 10))
	})

	It("Should allow disconnecting handlers", func(ctx SpecContext) {
		obs := observe.New[int]()
		called := false
		disconnect := obs.OnChange(func(ctx context.Context, v int) {
			called = true
		})
		disconnect()
		obs.Notify(ctx, 5)
		Expect(called).To(BeFalse())
	})
})

var _ = Describe("Translator", func() {
	It("Should translate and notify when Translate returns true", func(ctx SpecContext) {
		base := observe.New[int]()
		translator := observe.Translator[int, string]{
			Observable: base,
			Translate: func(ctx context.Context, v int) (string, bool) {
				return "translated", true
			},
		}
		var result string
		translator.OnChange(func(ctx context.Context, v string) {
			result = v
		})
		base.Notify(ctx, 42)
		Expect(result).To(Equal("translated"))
	})

	It("Should not notify when Translate returns false", func(ctx SpecContext) {
		base := observe.New[int]()
		translator := observe.Translator[int, string]{
			Observable: base,
			Translate: func(ctx context.Context, v int) (string, bool) {
				return "", false
			},
		}
		called := false
		translator.OnChange(func(ctx context.Context, v string) {
			called = true
		})
		base.Notify(ctx, 42)
		Expect(called).To(BeFalse())
	})

	It("Should conditionally notify based on input", func(ctx SpecContext) {
		base := observe.New[int]()
		translator := observe.Translator[int, int]{
			Observable: base,
			Translate: func(ctx context.Context, v int) (int, bool) {
				if v > 10 {
					return v * 2, true
				}
				return 0, false
			},
		}
		var results []int
		translator.OnChange(func(ctx context.Context, v int) {
			results = append(results, v)
		})
		base.Notify(ctx, 5)
		base.Notify(ctx, 15)
		base.Notify(ctx, 3)
		base.Notify(ctx, 20)
		Expect(results).To(Equal([]int{30, 40}))
	})
})

var _ = Describe("Noop", func() {
	It("Should not call handlers", func() {
		var noop observe.Noop[int]
		called := false
		noop.OnChange(func(ctx context.Context, v int) {
			called = true
		})
		Expect(called).To(BeFalse())
	})
})

var _ = Describe("AsyncObserver", func() {
	var (
		obs    observe.Observer[int]
		sCtx   signal.Context
		cancel context.CancelFunc
	)
	BeforeEach(func() {
		sCtx, cancel = signal.Isolated()
		obs = observe.NewAsync[int](sCtx, signal.WithRetryOnPanic(100))
	})
	AfterEach(func() {
		cancel()
		Expect(errors.Skip(sCtx.Wait(), context.Canceled)).To(Succeed())
	})

	Describe("Notify", func() {
		It("Should deliver values to a single handler", func() {
			var received atomic.Int64
			obs.OnChange(func(_ context.Context, v int) {
				received.Add(int64(v))
			})
			obs.Notify(context.Background(), 7)
			Eventually(received.Load).Should(Equal(int64(7)))
		})

		It("Should deliver values to multiple handlers", func() {
			var a, b atomic.Int64
			obs.OnChange(func(_ context.Context, v int) { a.Add(int64(v)) })
			obs.OnChange(func(_ context.Context, v int) { b.Add(int64(v)) })
			obs.Notify(context.Background(), 3)
			Eventually(a.Load).Should(Equal(int64(3)))
			Eventually(b.Load).Should(Equal(int64(3)))
		})

		It("Should not block when a handler is slow", func() {
			gate := make(chan struct{})
			obs.OnChange(func(_ context.Context, _ int) { <-gate })
			defer close(gate)

			done := make(chan struct{})
			go func() {
				defer close(done)
				for i := 0; i < 200; i++ {
					obs.Notify(context.Background(), i)
				}
			}()
			Eventually(done).Should(BeClosed())
		})

		It("Should drop values when a handler's buffer is full", func() {
			gate := make(chan struct{})
			var received atomic.Int64
			obs.OnChange(func(_ context.Context, _ int) {
				<-gate
				received.Add(1)
			})

			for i := 0; i < 200; i++ {
				obs.Notify(context.Background(), i)
			}
			close(gate)

			Eventually(func() int64 { return received.Load() }).
				Should(BeNumerically(">", 0))
			Eventually(func() int64 { return received.Load() }).
				Should(BeNumerically("<=", 65))
		})

		It("Should not block other handlers when one is slow", func() {
			gate := make(chan struct{})
			obs.OnChange(func(_ context.Context, _ int) { <-gate })
			defer close(gate)

			var fastCount atomic.Int64
			obs.OnChange(func(_ context.Context, _ int) { fastCount.Add(1) })

			for i := 0; i < 10; i++ {
				obs.Notify(context.Background(), i)
			}
			Eventually(fastCount.Load).Should(Equal(int64(10)))
		})

		It("Should pass the context through to handlers", func() {
			type ctxKey struct{}
			var received atomic.Value
			obs.OnChange(func(ctx context.Context, _ int) {
				received.Store(ctx.Value(ctxKey{}))
			})
			ctx := context.WithValue(context.Background(), ctxKey{}, "hello")
			obs.Notify(ctx, 1)
			Eventually(func() any { return received.Load() }).Should(Equal("hello"))
		})
	})

	Describe("NotifyGenerator", func() {
		It("Should call the generator once per handler", func() {
			var callCount atomic.Int64
			generator := func() int {
				return int(callCount.Add(1))
			}
			var a, b atomic.Int64
			obs.OnChange(func(_ context.Context, v int) { a.Store(int64(v)) })
			obs.OnChange(func(_ context.Context, v int) { b.Store(int64(v)) })
			obs.NotifyGenerator(context.Background(), generator)
			Eventually(func() int64 { return a.Load() + b.Load() }).
				Should(BeNumerically(">", 0))
			Expect(callCount.Load()).To(Equal(int64(2)))
		})

		It("Should generate unique values per handler", func() {
			counter := 0
			generator := func() int {
				counter++
				return counter
			}
			var a, b atomic.Int64
			obs.OnChange(func(_ context.Context, v int) { a.Store(int64(v)) })
			obs.OnChange(func(_ context.Context, v int) { b.Store(int64(v)) })
			obs.NotifyGenerator(context.Background(), generator)
			Eventually(func() bool {
				return a.Load() > 0 && b.Load() > 0
			}).Should(BeTrue())
			Expect(a.Load()).ToNot(Equal(b.Load()))
		})

		It("Should not block when a handler is slow", func() {
			gate := make(chan struct{})
			obs.OnChange(func(_ context.Context, _ int) { <-gate })
			defer close(gate)

			done := make(chan struct{})
			go func() {
				defer close(done)
				for i := 0; i < 200; i++ {
					obs.NotifyGenerator(context.Background(), func() int { return i })
				}
			}()
			Eventually(done).Should(BeClosed())
		})
	})

	Describe("GoNotify", func() {
		It("Should deliver values asynchronously", func() {
			var received atomic.Int64
			obs.OnChange(func(_ context.Context, v int) {
				received.Store(int64(v))
			})
			obs.GoNotify(context.Background(), 42)
			Eventually(received.Load).Should(Equal(int64(42)))
		})
	})

	Describe("Disconnect", func() {
		It("Should stop delivering values after disconnect", func() {
			var count atomic.Int64
			disconnect := obs.OnChange(func(_ context.Context, _ int) {
				count.Add(1)
			})
			obs.Notify(context.Background(), 1)
			Eventually(count.Load).Should(Equal(int64(1)))

			disconnect()
			obs.Notify(context.Background(), 2)
			Consistently(count.Load, 50*time.Millisecond).Should(Equal(int64(1)))
		})

		It("Should wait for in-flight handler to finish", func() {
			started := make(chan struct{})
			gate := make(chan struct{})
			var finished atomic.Bool
			disconnect := obs.OnChange(func(_ context.Context, _ int) {
				close(started)
				<-gate
				finished.Store(true)
			})
			obs.Notify(context.Background(), 1)
			<-started

			done := make(chan struct{})
			go func() {
				disconnect()
				close(done)
			}()

			Consistently(done, 50*time.Millisecond).ShouldNot(BeClosed())
			Expect(finished.Load()).To(BeFalse())

			close(gate)
			Eventually(done).Should(BeClosed())
			Expect(finished.Load()).To(BeTrue())
		})

		It("Should be safe to call multiple times", func() {
			disconnect := obs.OnChange(func(_ context.Context, _ int) {})
			disconnect()
			Expect(func() { disconnect() }).ToNot(Panic())
		})

		It("Should only stop the disconnected handler", func() {
			var a, b atomic.Int64
			disconnectA := obs.OnChange(func(_ context.Context, _ int) { a.Add(1) })
			obs.OnChange(func(_ context.Context, _ int) { b.Add(1) })

			disconnectA()
			obs.Notify(context.Background(), 1)
			Eventually(b.Load).Should(Equal(int64(1)))
			Consistently(a.Load, 50*time.Millisecond).Should(Equal(int64(0)))
		})
	})

	Describe("Concurrency", func() {
		It("Should handle concurrent Notify and OnChange", func() {
			var total atomic.Int64
			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				defer wg.Done()
				for i := 0; i < 100; i++ {
					obs.Notify(context.Background(), 1)
				}
			}()

			go func() {
				defer wg.Done()
				for i := 0; i < 50; i++ {
					d := obs.OnChange(func(_ context.Context, v int) {
						total.Add(int64(v))
					})
					d()
				}
			}()

			wg.Wait()
		})

		It("Should handle concurrent Notify and Disconnect", func() {
			var wg sync.WaitGroup
			disconnect := obs.OnChange(func(_ context.Context, _ int) {})
			wg.Add(2)

			go func() {
				defer wg.Done()
				for i := 0; i < 1000; i++ {
					obs.Notify(context.Background(), i)
				}
			}()

			go func() {
				defer wg.Done()
				time.Sleep(1 * time.Millisecond)
				disconnect()
			}()

			wg.Wait()
		})
	})

	Describe("Cancelled context", func() {
		It("Should not deadlock disconnect when context is already cancelled", func() {
			preCtx, preCancel := signal.Isolated()
			preCancel()
			Expect(errors.Skip(preCtx.Wait(), context.Canceled)).To(Succeed())
			preObs := observe.NewAsync[int](preCtx, signal.WithRetryOnPanic(100))
			disconnect := preObs.OnChange(func(_ context.Context, _ int) {})
			done := make(chan struct{})
			go func() {
				disconnect()
				close(done)
			}()
			Eventually(done, 500*time.Millisecond).Should(BeClosed())
		})
	})

	Describe("Panic recovery", func() {
		It("Should survive a handler panic and keep processing", func() {
			var count atomic.Int64
			obs.OnChange(func(_ context.Context, v int) {
				if v == 1 {
					panic("test panic")
				}
				count.Add(1)
			})
			obs.Notify(context.Background(), 1)
			obs.Notify(context.Background(), 2)
			obs.Notify(context.Background(), 3)
			Eventually(func() int64 { return count.Load() }).
				Should(BeNumerically(">=", 1))
		})

		It("Should not burn retries on disconnect after panic", func() {
			slowCtx, slowCancel := signal.Isolated()
			defer slowCancel()
			slowObs := observe.NewAsync[int](
				slowCtx,
				signal.WithRetryOnPanic(5),
				signal.WithBaseRetryInterval(50*time.Millisecond),
			)
			disconnect := slowObs.OnChange(func(_ context.Context, v int) {
				if v == 0 {
					panic("test panic")
				}
			})
			slowObs.Notify(context.Background(), 0)
			time.Sleep(10 * time.Millisecond)
			disconnect()
			Eventually(slowCtx.Stopped(), 2*time.Second).
				Should(BeClosed())
		})
	})

	Describe("Zero handlers", func() {
		It("Should not panic when notifying with no handlers", func() {
			Expect(func() {
				obs.Notify(context.Background(), 1)
			}).ToNot(Panic())
		})

		It("Should not panic when calling NotifyGenerator with no handlers",
			func() {
				Expect(func() {
					obs.NotifyGenerator(
						context.Background(),
						func() int { return 1 },
					)
				}).ToNot(Panic())
			})
	})
})
