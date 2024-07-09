// Copyright 2023 Synnax Labs, Inc.
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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
	"runtime/pprof"
	"time"
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

		Describe("CancelOnExitErr", func() {
			It("Should cancel the context when the first routine exits with an error", func() {
				ctx, cancel := signal.Isolated()
				ctx.Go(immediatelyReturnNil, signal.CancelOnExitErr())
				Expect(ctx.Stopped()).ToNot(BeClosed())
				ctx.Go(immediatelyReturnError, signal.CancelOnExitErr())
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
			}, signal.WithPanicPolicy(signal.RecoverErr))

			Expect(ctx.Wait()).To(MatchError(ContainSubstring("routine panicked")))
		})

		It("Should not error when instructed to not error", func() {
			ctx, _ := signal.Isolated()
			ctx.Go(func(ctx context.Context) error {
				return immediatelyPanic(ctx)
			}, signal.WithPanicPolicy(signal.RecoverNoErr))

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
			ctx.Go(
				inc1,
				signal.WithPanicPolicy(signal.Restart),
				signal.WithMaxRestart(100),
			)

			Expect(ctx.Wait()).To(MatchError(ContainSubstring("panicking once")))
			Expect(counter).To(Equal(101))
		})

	})

})
