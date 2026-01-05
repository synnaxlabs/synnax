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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Switch", func() {
	Describe("Switch", func() {
		Context("Single Inlet", func() {
			var (
				ctx    signal.Context
				cancel context.CancelFunc
				input  *Stream[int]
				double *Stream[int]
				single *Stream[int]
				sw     *Switch[int]
			)
			BeforeEach(func() {
				ctx, cancel = signal.Isolated()
				input = NewStream[int](3)
				double = NewStream[int](3)
				single = NewStream[int](3)
				double.SetInletAddress("double")
				single.SetInletAddress("single")
				sw = &Switch[int]{}
				sw.InFrom(input)
				sw.OutTo(double)
				sw.OutTo(single)
			})
			AfterEach(func() {
				cancel()
			})
			It("Should route values to the correct inlets", func() {
				sw.Switch = func(ctx context.Context, i int) (address.Address, bool, error) {
					if i%2 == 0 {
						return "single", true, nil
					} else {
						return "double", true, nil
					}
				}
				sw.Flow(ctx, CloseOutputInletsOnExit())
				input.Inlet() <- 1
				input.Inlet() <- 2
				input.Inlet() <- 3
				input.Close()
				Expect(ctx.Wait()).To(Succeed())
				Expect(<-double.Outlet()).To(Equal(1))
				Expect(<-single.Outlet()).To(Equal(2))
				Expect(<-double.Outlet()).To(Equal(3))
				_, ok := <-double.Outlet()
				Expect(ok).To(BeFalse())
			})
			It("Should exit of the switch returns an error", func() {
				sw.Switch = func(ctx context.Context, i int) (address.Address, bool, error) {
					return "", false, errors.New("test error")
				}
				sw.Flow(ctx, CloseOutputInletsOnExit())
				input.Inlet() <- 1
				input.Inlet() <- 2
				input.Inlet() <- 3
				input.Close()
				Expect(ctx.Wait()).To(MatchError("test error"))
			})
			It("Should return an error if the address can't be resolved", func() {
				sw.Switch = func(ctx context.Context, i int) (address.Address, bool, error) {
					return "hello", true, nil
				}
				sw.Flow(ctx, CloseOutputInletsOnExit(), WithAddress("toCoverThis"))
				input.Inlet() <- 1
				Expect(ctx.Wait()).To(MatchError(address.ErrNotFound))

			})
		})
	})
	Describe("BatchSwitch", func() {
		var (
			ctx    signal.Context
			cancel context.CancelFunc
			input  *Stream[[]int]
			first  *Stream[int]
			second *Stream[int]
			sw     *BatchSwitch[[]int, int]
		)
		BeforeEach(func() {
			ctx, cancel = signal.Isolated()
			input = NewStream[[]int](3)
			first = NewStream[int](3)
			first.SetInletAddress("first")
			second = NewStream[int](3)
			second.SetInletAddress("second")
			sw = &BatchSwitch[[]int, int]{}
			sw.InFrom(input)
			sw.OutTo(first)
			sw.OutTo(second)
		})
		It("Should route values to the correct inlets", func() {
			sw.Switch = func(
				ctx context.Context,
				i []int,
				o map[address.Address]int,
			) error {
				// first to first, second to second
				o["first"] = i[0]
				o["second"] = i[1]
				if i[0] == 5 {
					return errors.New("error")
				}
				return nil
			}
			sw.Flow(ctx, CloseOutputInletsOnExit())
			input.Inlet() <- []int{1, 2}
			input.Inlet() <- []int{3, 4}
			input.Inlet() <- []int{5, 6}
			Expect(ctx.Wait()).ToNot(Succeed())
			Expect(<-first.Outlet()).To(Equal(1))
			Expect(<-second.Outlet()).To(Equal(2))
			Expect(<-first.Outlet()).To(Equal(3))
			Expect(<-second.Outlet()).To(Equal(4))
			_, ok := <-first.Outlet()
			Expect(ok).To(BeFalse())
			_, ok = <-second.Outlet()
			Expect(ok).To(BeFalse())
		})
		It("Should exit if the context is cancelled", func() {
			sw.Switch = func(
				ctx context.Context,
				i []int,
				o map[address.Address]int,
			) error {
				// first to first, second to second
				o["first"] = i[0]
				o["second"] = i[1]
				return nil
			}
			sw.Flow(ctx, CloseOutputInletsOnExit())
			input.Inlet() <- []int{1, 2}
			input.Inlet() <- []int{3, 4}
			input.Inlet() <- []int{5, 6}
			cancel()
			Expect(ctx.Wait()).To(HaveOccurredAs(context.Canceled))
		})

	})
})
