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
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
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
				ctx, cancel := signal.TODO()
				ctx.Go(immediatelyReturnNil, signal.CancelOnExit())
				ctx.Go(returnErrAfterContextCancel)
				cancel()
				Expect(ctx.Wait()).To(HaveOccurredAs(context.Canceled))
				Eventually(ctx.Stopped()).Should(BeClosed())
			})
		})

		Describe("CancelOnExitErr", func() {
			It("Should cancel the context when the first routine exits with an error", func() {
				ctx, cancel := signal.TODO()
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
				ctx, cancel := signal.TODO()
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
				ctx, cancel := signal.TODO()
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
				ctx, cancel := signal.TODO()
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
				ctx, cancel := signal.TODO()
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
				ctx, cancel := signal.TODO()
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
				ctx, cancel := signal.TODO()
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

	Describe("Options", func() {

		Describe("WithInstrumentation", func() {
			It("Should inject a instrumentation into the context for diagnostics", func() {
				ctx, cancel := signal.TODO(signal.WithInstrumentation(zap.NewNop()))
				cancel()
				Expect(ctx.Err()).To(HaveOccurredAs(context.Canceled))
			})
		})

	})

	Describe("Profiler Labels", func() {

		It("Should add a profiler label with the routine key", func() {
			ctx, cancel := signal.TODO()
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

		It("Should prefix the routine key with the context key", func() {
			ctx, cancel := signal.Background(signal.WithContextKey("context-1"))
			defer cancel()
			ctx.Go(func(ctx context.Context) error {
				defer GinkgoRecover()
				v, ok := pprof.Label(ctx, "routine")
				Expect(ok).To(BeTrue())
				Expect(v).To(Equal("context-1.routine-1"))
				return nil
			}, signal.WithKey("routine-1"))
			Expect(ctx.Wait()).To(Succeed())
		})

	})

	Describe("Census", func() {

		It("Should return the routines running under the context", func() {
			ctx, cancel := signal.TODO()
			defer cancel()
			ctx.Go(immediatelyReturnNil)
			Expect(ctx.Wait()).ToNot(HaveOccurred())
			Expect(ctx.Routines()).To(HaveLen(1))
		})

	})

	Describe("SendUnderContext", func() {

		It("Should send a value to the channel", func() {
			v := make(chan int, 1)
			signal.SendUnderContext(context.Background(), v, 1)
			Expect(<-v).To(Equal(1))
		})

		It("Should not send a value to the channel if the context is cancelled", func() {
			ctx, cancel := signal.WithTimeout(context.TODO(), 500*time.Microsecond)
			v := make(chan int)
			signal.SendUnderContext(ctx, v, 1)
			cancel()
			Expect(v).ToNot(Receive())
		})

	})

})
