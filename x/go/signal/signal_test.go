// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signal_test

import (
	"context"
	"runtime/pprof"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/atomic"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
)

func immediatelyReturnError(ctx context.Context) error {
	return errors.New("routine failed")
}

func immediatelyPanic(ctx context.Context) error {
	panic("routine panicked")
}

func immediatelyReturnNil(ctx context.Context) error {
	return nil
}

func returnErrAfterContextCancel(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

var _ = Describe("Signal", func() {

	Describe("Coordination", func() {

		Describe("CancelOnExit", func() {
			It("Should cancel the context when the first routine exits", func() {
				ctx, cancel := signal.Isolated()
				ctx.Go(immediatelyReturnNil, signal.CancelOnExit())
				ctx.Go(returnErrAfterContextCancel)
				cancel()
				Expect(ctx.Wait()).To(HaveOccurredAs(context.Canceled))
				Eventually(ctx.Stopped()).Should(BeClosed())
			})
		})

		Describe("CancelOnFail", func() {
			It("Should cancel the context when the first routine exits with an error", func() {
				ctx, cancel := signal.Isolated()
				ctx.Go(immediatelyReturnNil, signal.CancelOnFail())
				Expect(ctx.Stopped()).ToNot(BeClosed())
				ctx.Go(immediatelyReturnError, signal.CancelOnFail())
				cancel()
				Expect(ctx.Wait()).To(HaveOccurredAs(errors.New("routine failed")))
				Eventually(ctx.Stopped()).Should(BeClosed())
			})
		})

		Context("Context already cancelled", func() {
			It("Shouldn't start a new routine", func() {
				ctx, cancel := signal.Isolated()
				cancel()
				c := 0
				ctx.Go(func(ctx context.Context) error {
					c++
					return nil
				})
				Expect(ctx.Wait()).To(Succeed())
				Expect(c).To(Equal(0))
			})

		})

	})

	Describe("Go Utilities", func() {

		Describe("GoRange", func() {

			It("Should range over a channel until the context is cancelled", func() {
				v := make(chan int, 3)
				ctx, cancel := signal.Isolated()
				c := 0
				signal.GoRange(ctx, v, func(ctx context.Context, v int) error {
					c++
					return nil
				})
				v <- 1
				v <- 2
				Eventually(func() int { return c }).Should(Equal(2))
				cancel()
				v <- 3
				Expect(ctx.Wait()).To(HaveOccurredAs(context.Canceled))
				Eventually(ctx.Stopped()).Should(BeClosed())
			})

			It("Should exit when the channel is closed", func() {
				v := make(chan int, 3)
				ctx, cancel := signal.Isolated()
				defer cancel()
				c := 0
				signal.GoRange(ctx, v, func(ctx context.Context, v int) error {
					c++
					return nil
				})
				v <- 1
				v <- 2
				Eventually(func() int { return c }).Should(Equal(2))
				close(v)
				Expect(ctx.Wait()).ToNot(HaveOccurred())
				Eventually(ctx.Stopped()).Should(BeClosed())
			})

			It("Should exit if the function returns a non-nil error", func() {
				v := make(chan int, 3)
				ctx, cancel := signal.Isolated()
				defer cancel()
				c := 0
				signal.GoRange(ctx, v, func(ctx context.Context, v int) error {
					c++
					return errors.New("routine failed")
				})
				v <- 1
				v <- 2
				Eventually(func() int { return c }).Should(Equal(1))
				Expect(ctx.Wait()).To(HaveOccurredAs(errors.New("routine failed")))
				Eventually(ctx.Stopped()).Should(BeClosed())
			})

		})

		Describe("GoTick", func() {
			It("Should tick until the context is cancelled", func() {
				ctx, cancel := signal.Isolated()
				defer cancel()
				c := 0
				signal.GoTick(ctx, 500*time.Microsecond, func(ctx context.Context, t time.Time) error {
					c++
					return nil
				})
				Eventually(func() int { return c }).Should(BeNumerically(">", 3))
				cancel()
				Expect(ctx.Wait()).To(HaveOccurredAs(context.Canceled))
				Eventually(ctx.Stopped()).Should(BeClosed())
			})
		})

	})

	Describe("GoOptions", func() {

		Describe("Defer", func() {
			It("Should defer a function until the routine exit", func() {
				ctx, cancel := signal.Isolated()
				defer cancel()
				c := 0
				ctx.Go(immediatelyReturnNil, signal.Defer(func() {
					c++
				}))
				Eventually(func() int { return c }).Should(Equal(1))
				Expect(ctx.Wait()).ToNot(HaveOccurred())
				Eventually(ctx.Stopped()).Should(BeClosed())
			})
		})

	})

	Describe("Profiler Labels", func() {

		It("Should add a profiler label with the routine key", func() {
			ctx, cancel := signal.Isolated()
			defer cancel()
			ctx.Go(func(ctx context.Context) error {
				defer GinkgoRecover()
				v, ok := pprof.Label(ctx, "routine")
				Expect(ok).To(BeTrue())
				Expect(v).To(Equal("routine-1"))
				return nil
			}, signal.WithKey("routine-1"))
			Expect(ctx.Wait()).To(Succeed())
		})

	})

	Describe("Census", func() {

		It("Should return the routines running under the context", func() {
			ctx, cancel := signal.Isolated()
			defer cancel()
			ctx.Go(immediatelyReturnNil)
			Expect(ctx.Wait()).ToNot(HaveOccurred())
			Expect(ctx.Routines()).To(HaveLen(1))
		})

	})

	Describe("SendUnderContext", func() {

		It("Should send a value to the channel", func() {
			v := make(chan int, 1)
			_ = signal.SendUnderContext(context.Background(), v, 1)
			Expect(<-v).To(Equal(1))
		})

		It("Should not send a value to the channel if the context is cancelled", func() {
			ctx, cancel := signal.WithTimeout(context.TODO(), 500*time.Microsecond)
			v := make(chan int)
			_ = signal.SendUnderContext(ctx, v, 1)
			cancel()
			Expect(v).ToNot(Receive())
		})

	})

	Describe("RecvUnderContext", func() {

		It("Should receive a value from the channel", func() {
			v := make(chan int, 1)
			v <- 1
			val, err := signal.RecvUnderContext(context.Background(), v)
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal(1))
		})

		It("Should return context error if context is cancelled before receive", func() {
			ctx, cancel := signal.WithTimeout(context.TODO(), 500*time.Microsecond)
			v := make(chan int)
			cancel()
			val, err := signal.RecvUnderContext(ctx, v)
			Expect(err).To(HaveOccurredAs(context.Canceled))
			Expect(val).To(Equal(0))
		})

		It("Should receive value even if context is cancelled after value is available", func() {
			ctx, cancel := signal.WithTimeout(context.TODO(), 500*time.Microsecond)
			v := make(chan int, 1)
			v <- 1
			val, err := signal.RecvUnderContext(ctx, v)
			cancel()
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal(1))
		})

	})

	Describe("Panic recovery", func() {

		// We cannot test with a test case that a goroutine indeed panics when it is
		// instructed to propagate its panic since there is no way to capture a panic
		// in another goroutine. However, we have manually tested that it indeed
		// panics the whole program.
		// We can test all other cases where panics are recovered.

		It("Should error a panic when instructed", func() {
			ctx, _ := signal.Isolated()
			ctx.Go(func(ctx context.Context) error {
				return immediatelyPanic(ctx)
			}, signal.RecoverWithErrOnPanic())

			Expect(ctx.Wait()).To(MatchError(ContainSubstring("routine panicked")))
		})

		It("Should not error when instructed to not error", func() {
			ctx, _ := signal.Isolated()
			ctx.Go(func(ctx context.Context) error {
				return immediatelyPanic(ctx)
			}, signal.RecoverWithoutErrOnPanic())

			Expect(ctx.Wait()).To(BeNil())
		})

		It("Should try to restart when instructed to", func() {
			var (
				counter = 0
				inc1    = func(ctx context.Context) error {
					counter += 1
					panic("panicking once")
				}
			)

			ctx, _ := signal.Isolated()
			ctx.Go(inc1, signal.WithBreaker(breaker.Config{MaxRetries: 100, BaseInterval: 1 * time.Millisecond, Scale: 1.01}), signal.RecoverWithErrOnPanic())

			Expect(ctx.Wait()).To(MatchError(ContainSubstring("panicking once")))
			Expect(counter).To(Equal(101))
		})

		It("Should try to restart when instructed to and follow the panic policy", func() {
			var (
				counter = 0
				inc1    = func(ctx context.Context) error {
					counter += 1
					panic("panicking once")
				}
			)

			ctx, _ := signal.Isolated()
			ctx.Go(inc1, signal.WithBreaker(breaker.Config{MaxRetries: 100, BaseInterval: 1 * time.Millisecond, Scale: 1.01}), signal.RecoverWithoutErrOnPanic())

			Expect(ctx.Wait()).ToNot(HaveOccurred())
			Expect(counter).To(Equal(101))
		})

		It("Should indefinitely restart", func() {
			var (
				done = make(chan struct{})
				wg   = sync.WaitGroup{}
				f    = func(ctx context.Context) error {
					select {
					case <-ctx.Done():
						return nil
					default:
						panic("panicking")
					}
				}
			)

			ctx, cancel := signal.Isolated()
			ctx.Go(f, signal.WithBreaker(breaker.Config{MaxRetries: breaker.InfiniteRetries, BaseInterval: 1 * time.Millisecond, Scale: 1.01}))

			wg.Add(1)
			go func() {
				defer wg.Done()
				Expect(ctx.Wait()).To(Succeed())
				close(done)
			}()

			cancel()
			wg.Wait()
			Eventually(done).Should(BeClosed())
		})

		It("Should wait exponentially more time", func() {
			var (
				done         = make(chan struct{})
				counter      = atomic.Int64Counter{}
				succeedInTen = func(ctx context.Context) error {
					if counter.Add(1) < 10 {
						panic("panicking")
					}
					return nil
				}
			)

			ctx, _ := signal.Isolated()
			start := time.Now()
			ctx.Go(
				succeedInTen,
				signal.WithBreaker(breaker.Config{MaxRetries: breaker.InfiniteRetries, BaseInterval: 1 * time.Millisecond, Scale: 2}),
				signal.RecoverWithErrOnPanic(),
			)

			go func() {
				Expect(ctx.Wait()).ToNot(HaveOccurred())
				close(done)
			}()

			Eventually(done).Should(BeClosed())
			Expect(time.Since(start)).To(BeNumerically("~", 511*time.Millisecond, 150*time.Millisecond))
		})

	})

	Describe("Regression", func() {
		// This test was added to address the bug where if maxRestart is set, even in
		// the case where the goroutine did not panic, it would attempt to restart.
		// This is not the desired behaviour since a goroutine should not attempt to
		// restart if it did not panic.
		It("Should NOT restart if there was not a panic - definite restart", func() {
			var (
				counter = 0
				f       = func(ctx context.Context) error {
					counter += 1
					return nil
				}
			)

			ctx, _ := signal.Isolated()
			ctx.Go(f, signal.WithRetryOnPanic(100))

			Expect(ctx.Wait()).To(Succeed())
			Expect(counter).To(Equal(1))
		})

		It("Should NOT restart if there was not a panic - infinite restart", func() {
			var (
				counter = 0
				f       = func(ctx context.Context) error {
					counter += 1
					return nil
				}
			)

			ctx, _ := signal.Isolated()
			ctx.Go(f, signal.WithRetryOnPanic())

			Expect(ctx.Wait()).To(Succeed())
			Expect(counter).To(Equal(1))
		})
	})

	Describe("Shutdown Closers", func() {
		It("NewHardShutdown should cancel context and wait for routines, skipping context.Canceled error", func() {
			ctx, cancel := signal.Isolated()
			done := make(chan struct{})
			ctx.Go(func(ctx context.Context) error {
				<-ctx.Done()
				close(done)
				return ctx.Err()
			})
			closer := signal.NewHardShutdown(ctx, cancel)
			err := closer.Close()
			Expect(err).To(BeNil()) // context.Canceled should be skipped
			Eventually(done).Should(BeClosed())
			Eventually(ctx.Stopped()).Should(BeClosed())
		})

		It("NewGracefulShutdown should wait for routines and then cancel context", func() {
			ctx, cancel := signal.Isolated()
			release := make(chan struct{})
			exit := make(chan struct{})
			ctx.Go(func(ctx context.Context) error {
				select {
				case <-release:
					close(exit)
					return nil
				case <-ctx.Done():
				}
				return ctx.Err()
			})
			closer := signal.NewGracefulShutdown(ctx, cancel)
			close(release)
			err := closer.Close()
			Eventually(exit).Should(BeClosed())
			Expect(err).To(BeNil())
			Eventually(ctx.Stopped()).Should(BeClosed())
		})
	})

})
