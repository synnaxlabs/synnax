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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	. "github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

var _ = Describe("Delta", func() {
	var (
		inputOne  *Stream[int]
		outputOne *Stream[int]
		outputTwo *Stream[int]
	)
	BeforeEach(func() {
		inputOne = NewStream[int](1)
		outputOne = NewStream[int](0)
		outputOne.SetInletAddress("outputOne")
		outputTwo = NewStream[int](0)
		outputTwo.SetInletAddress("outputTwo")
	})

	Describe("DeltaMultiplier", func() {
		It("Should multiply input values to outputs", func() {
			delta := &DeltaMultiplier[int]{}
			delta.OutTo(outputOne, outputTwo)
			delta.InFrom(inputOne)
			ctx, cancel := signal.Isolated()
			defer cancel()
			delta.Flow(ctx)
			inputOne.Inlet() <- 1
			v1 := <-outputOne.Outlet()
			v2 := <-outputTwo.Outlet()
			Expect(v1).To(Equal(1))
			Expect(v2).To(Equal(1))
		})
		It("Should close inlets when the delta is closed", func() {
			delta := &DeltaMultiplier[int]{}
			delta.OutTo(outputOne)
			delta.InFrom(inputOne)
			ctx, cancel := signal.Isolated()
			defer cancel()
			delta.Flow(ctx, CloseOutputInletsOnExit())
			inputOne.Inlet() <- 1
			inputOne.Close()
			v1 := <-outputOne.Outlet()
			Expect(v1).To(Equal(1))
			_, ok := <-outputOne.Outlet()
			Expect(ok).To(BeFalse())
		})
	})

	Describe("DeltaTransformMultiplier", func() {
		It("Should multiply input values to outputs", func() {
			delta := &DeltaTransformMultiplier[int, int]{}
			delta.Transform = func(ctx context.Context, v int) (int, bool, error) {
				return v * 2, true, nil
			}
			delta.OutTo(outputOne, outputTwo)
			delta.InFrom(inputOne)
			ctx, cancel := signal.Isolated()
			defer cancel()
			delta.Flow(ctx)
			inputOne.Inlet() <- 1
			v1 := <-outputOne.Outlet()
			v2 := <-outputTwo.Outlet()
			Expect(v1).To(Equal(2))
			Expect(v2).To(Equal(2))
		})
		It("Should close inlets when the delta is closed", func() {
			delta := &DeltaTransformMultiplier[int, int]{}
			delta.Transform = func(ctx context.Context, v int) (int, bool, error) {
				return v * 2, true, nil
			}
			delta.OutTo(outputOne)
			delta.InFrom(inputOne)
			ctx, cancel := signal.Isolated()
			defer cancel()
			delta.Flow(ctx, CloseOutputInletsOnExit())
			inputOne.Inlet() <- 1
			inputOne.Close()
			v1 := <-outputOne.Outlet()
			Expect(v1).To(Equal(2))
			_, ok := <-outputOne.Outlet()
			Expect(ok).To(BeFalse())
		})
		It("Should not send a value when the transform returns false", func() {
			delta := &DeltaTransformMultiplier[int, int]{}
			delta.Transform = func(ctx context.Context, v int) (int, bool, error) {
				return v * 2, v != 1, nil
			}
			delta.OutTo(outputOne)
			delta.InFrom(inputOne)
			ctx, cancel := signal.Isolated()
			defer cancel()
			delta.Flow(ctx, CloseOutputInletsOnExit())
			inputOne.Inlet() <- 1
			inputOne.Close()
			_, ok := <-outputOne.Outlet()
			Expect(ok).To(BeFalse())
		})
	})

	Describe("DynamicDeltaMultiplier", func() {
		It("Should allow the caller to add and remove outlets dynamically", func() {
			delta := NewDynamicDeltaMultiplier[int](0, Instrumentation("dev"))
			delta.InFrom(inputOne)
			ctx, cancel := signal.Isolated()
			defer cancel()
			delta.Flow(ctx, CloseOutputInletsOnExit())
			delta.Connect(outputOne)
			delta.Connect(outputTwo)
			Eventually(inputOne.Inlet()).Should(BeSent(1))
			Eventually(outputOne.Outlet()).Should(Receive(Equal(1)))
			Eventually(outputTwo.Outlet()).Should(Receive(Equal(1)))
			delta.Disconnect(outputOne)
			Eventually(inputOne.Inlet()).Should(BeSent(2))
			Eventually(outputTwo.Outlet()).Should(Receive(Equal(2)))
			Eventually(outputOne.Outlet()).Should(BeClosed())
		})

		Describe("Stopping", func() {
			var (
				delta  *DynamicDeltaMultiplier[int]
				ctx    signal.Context
				cancel context.CancelFunc
			)
			BeforeEach(func() {
				delta = NewDynamicDeltaMultiplier[int](0, Instrumentation("dev"))
				delta.InFrom(inputOne)
				ctx, cancel = signal.Isolated()
				DeferCleanup(cancel)
			})

			It("Should close connected inlets and report that it stopped", func() {
				delta.Flow(ctx)
				delta.Connect(outputOne)
				Consistently(delta.Stopped(), 10*time.Millisecond).ShouldNot(BeClosed())
				cancel()
				Eventually(delta.Stopped()).Should(BeClosed())
				Eventually(outputOne.Outlet()).Should(BeClosed())
			})

			It("Should stop when its input closes", func() {
				delta.Flow(ctx)
				inputOne.Close()
				Eventually(delta.Stopped()).Should(BeClosed())
			})

			It("Should close an inlet connected after it stopped", func() {
				delta.Flow(ctx)
				cancel()
				Eventually(delta.Stopped()).Should(BeClosed())
				connected := make(chan struct{})
				go func() {
					defer close(connected)
					delta.Connect(outputOne)
				}()
				Eventually(connected).Should(BeClosed())
				Eventually(outputOne.Outlet()).Should(BeClosed())
			})

			It("Should return from Disconnect after it stopped", func() {
				delta.Flow(ctx)
				delta.Connect(outputOne)
				cancel()
				Eventually(delta.Stopped()).Should(BeClosed())
				disconnected := make(chan struct{})
				go func() {
					defer close(disconnected)
					delta.Disconnect(outputOne)
				}()
				Eventually(disconnected).Should(BeClosed())
			})

			It("Should drop only the inlet that panics", func() {
				delta.Flow(
					ctx,
					RecoverWithErrOnPanic(),
					WithRetryOnPanic(),
					func(o *Options) {
						o.Signal = append(
							o.Signal,
							signal.WithBaseRetryInterval(time.Millisecond),
							signal.WithRetryScale(1e6),
						)
					},
				)
				delta.Connect(outputOne, outputTwo)
				outputTwo.Close()
				for v := range 5 {
					Eventually(inputOne.Inlet()).Should(BeSent(v))
					Eventually(outputOne.Outlet()).Should(Receive(Equal(v)))
				}
				Expect(delta.Stopped()).ToNot(BeClosed())
				cancel()
				Eventually(delta.Stopped()).Should(BeClosed())
			})
		})

		Describe("Timeout", func() {
			It(
				"Should allow the delta to operate normally if a consumer receives within the timeout",
				func() {
					delta := NewDynamicDeltaMultiplier[int](
						100*time.Millisecond,
						Instrumentation("dev", InstrumentationConfig{Log: new(true)}),
					)
					delta.InFrom(inputOne)
					ctx, cancel := signal.Isolated()
					defer cancel()
					delta.Flow(ctx, CloseOutputInletsOnExit())
					delta.Connect(outputOne)
					delta.Connect(outputTwo)
					Eventually(inputOne.Inlet()).Should(BeSent(1))
					Eventually(
						outputOne.Outlet(),
						time.Second,
						time.Millisecond,
					).Should(Receive(Equal(1)))
					Eventually(
						outputTwo.Outlet(),
						time.Second,
						time.Millisecond,
					).Should(Receive(Equal(1)))
					delta.Disconnect(outputOne)
					Eventually(inputOne.Inlet()).Should(BeSent(2))
					Eventually(
						outputTwo.Outlet(),
						time.Second,
						time.Millisecond,
					).Should(Receive(Equal(2)))
					Eventually(
						outputOne.Outlet(),
						time.Second,
						time.Millisecond,
					).Should(BeClosed())
				},
			)

			It(
				"Should allow other outlets to receive values even if one consumer times out",
				func() {
					delta := NewDynamicDeltaMultiplier[int](
						10*time.Millisecond,
						Instrumentation("dev", InstrumentationConfig{Log: new(false)}),
					)
					delta.InFrom(inputOne)
					ctx, cancel := signal.Isolated()
					defer cancel()
					delta.Flow(ctx, CloseOutputInletsOnExit())
					delta.Connect(outputOne)
					delta.Connect(outputTwo)
					Eventually(inputOne.Inlet()).Should(BeSent(1))
					Eventually(
						outputTwo.Outlet(),
						time.Millisecond*40,
						time.Millisecond,
					).Should(Receive(Equal(1)))
				},
			)
		})
	})
})
