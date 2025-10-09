// Copyright 2025 Synnax Labs, Inc.
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
	"runtime"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Emitter", func() {
	It("Should emit values at regular intervals", func() {
		e := &Emitter[int]{
			Interval: 1 * time.Millisecond,
			Emit:     func(context.Context) (int, error) { return 1, nil },
		}
		ctx, cancel := signal.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		stream := NewStream[int](0)
		e.OutTo(stream)
		e.Flow(ctx, CloseOutputInletsOnExit())
		var received []int
		for v := range stream.Outlet() {
			received = append(received, v)
			runtime.Gosched()
		}
		Expect(len(received)).To(BeNumerically(">", 0))
	})
	It("Should exit if the emitter returns an error", func() {
		e := &Emitter[int]{
			Interval: 1 * time.Millisecond,
			Emit: func(context.Context) (int, error) {
				return 1, errors.New("exited")
			},
		}
		ctx, cancel := signal.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		stream := NewStream[int](0)
		e.OutTo(stream)
		e.Flow(ctx, CloseOutputInletsOnExit())
		Expect(ctx.Wait()).To(HaveOccurredAs(errors.New("exited")))
		_, ok := <-stream.Outlet()
		Expect(ok).To(BeFalse())
	})
})
