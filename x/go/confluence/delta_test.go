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
	. "github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
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

		Describe("Timeout", func() {
			It("Should allow the delta to operate normally if a consumer receives within the timeout", func() {
				delta := NewDynamicDeltaMultiplier[int](
					10*time.Millisecond,
					Instrumentation("dev", InstrumentationConfig{Log: new(true)}),
				)
				delta.InFrom(inputOne)
				ctx, cancel := signal.Isolated()
				defer cancel()
				delta.Flow(ctx, CloseOutputInletsOnExit())
				delta.Connect(outputOne)
				delta.Connect(outputTwo)
				Eventually(inputOne.Inlet()).Should(BeSent(1))
				Eventually(outputOne.Outlet(), "1s", "1ms").Should(Receive(Equal(1)))
				Eventually(outputTwo.Outlet(), "1s", "1ms").Should(Receive(Equal(1)))
				delta.Disconnect(outputOne)
				Eventually(inputOne.Inlet()).Should(BeSent(2))
				Eventually(outputTwo.Outlet(), "1s", "1ms").Should(Receive(Equal(2)))
				Eventually(outputOne.Outlet(), "1s", "1ms").Should(BeClosed())
			})

			It("Should allow other outlets to receive values even if one consumer times out", func() {
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
				Eventually(outputTwo.Outlet(), "40ms", "1ms").Should(Receive(Equal(1)))
			})
		})
	})

})
