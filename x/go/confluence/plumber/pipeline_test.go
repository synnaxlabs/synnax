// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package plumber_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
)

var _ = Describe("Pipeline", func() {
	var pipe *plumber.Pipeline
	BeforeEach(func() { pipe = plumber.New() })
	Describe("Basic Usage", func() {
		It("Should set and get a source", func() {
			emitter := &confluence.Emitter[int]{}
			plumber.SetSource(pipe, "source", emitter)
			source, err := plumber.GetSource[int](pipe, "source")
			Expect(err).ToNot(HaveOccurred())
			Expect(source).To(Equal(emitter))
		})

		It("Should set and get a sink", func() {
			unarySink := &confluence.UnarySink[int]{}
			plumber.SetSink(pipe, "sink", unarySink)
			sink, err := plumber.GetSink[int](pipe, "sink")
			Expect(err).ToNot(HaveOccurred())
			Expect(sink).To(Equal(unarySink))
		})
	})

	Describe("NewHardShutdown Chain", func() {
		It("Should shutdown the pipe as segments close their inlets", func() {
			t1 := &confluence.LinearTransform[int, int]{}
			t1.Transform = func(_ context.Context, v int) (int, bool, error) {
				return v * 2, true, nil
			}
			plumber.SetSegment(pipe, "t1", t1, confluence.CloseOutputInletsOnExit())

			t2 := &confluence.LinearTransform[int, int]{}
			t2.Transform = func(_ context.Context, v int) (int, bool, error) {
				return v * 2, true, nil
			}
			plumber.SetSegment(pipe, "t2", t2, confluence.CloseOutputInletsOnExit())

			Expect(plumber.UnaryRouter[int]{
				SourceTarget: "t1",
				SinkTarget:   "t2",
			}.Route(pipe)).To(Succeed())

			seg := &plumber.Segment[int, int]{Pipeline: pipe}
			Expect(seg.RouteInletTo("t1")).To(Succeed())
			Expect(seg.RouteOutletFrom("t2")).To(Succeed())

			input := confluence.NewStream[int](1)
			output := confluence.NewStream[int](0)
			seg.InFrom(input)
			seg.OutTo(output)

			ctx, cancel := signal.Isolated()
			defer cancel()
			seg.Flow(ctx)

			input.Inlet() <- 1
			input.Close()
			v := <-output.Outlet()
			Expect(v).To(Equal(4))
			_, ok := <-output.Outlet()
			Expect(ok).To(BeFalse())
		})
	})

	Describe("GetSink", func() {
		It("Should return an error if the sink is not found", func() {
			_, err := plumber.GetSink[int](pipe, "sink")
			Expect(err).To(HaveOccurred())
		})
		It("Should return an error if the sink is of the wrong type", func() {
			plumber.SetSink[int](pipe, "sink", &confluence.UnarySink[int]{})
			_, err := plumber.GetSink[[]int](pipe, "sink")
			Expect(err).To(HaveOccurred())
		})
	})
	Describe("GetSource", func() {
		It("Should return an error if the source is not found", func() {
			_, err := plumber.GetSource[int](pipe, "source")
			Expect(err).To(HaveOccurred())
		})
		It("Should return an error if the sink is of the wrong type", func() {
			plumber.SetSource(pipe, "sink", &confluence.Emitter[int]{})
			_, err := plumber.GetSink[[]int](pipe, "sink")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Complex Pipeline", func() {
		It("Should construct and operate the pipe correctly", func() {
			emitterOne := &confluence.Emitter[int]{Interval: 1 * time.Millisecond}
			c1 := 0
			emitterOne.Emit = func(context.Context) (int, error) {
				c1++
				if c1 == 5 {
					return 0, errors.New("done counting")
				}
				return c1, nil
			}
			plumber.SetSource(pipe, "emitterOne", emitterOne)

			emitterTwo := &confluence.Emitter[int]{Interval: 1 * time.Millisecond}
			c2 := 0
			emitterTwo.Emit = func(context.Context) (int, error) {
				c2++
				if c2 == 6 {
					return 0, errors.New("done counting")
				}
				return c2, nil
			}
			plumber.SetSource(pipe, "emitterTwo", emitterTwo)

			t1 := &confluence.LinearTransform[int, int]{}
			t1.Transform = func(_ context.Context, v int) (int, bool, error) {
				return v * 2, true, nil
			}
			plumber.SetSegment(pipe, "t1", t1)

			t2 := &confluence.LinearTransform[int, int]{}
			t2.Transform = func(_ context.Context, v int) (int, bool, error) {
				return v * 3, true, nil
			}
			plumber.SetSegment(pipe, "t2", t2)

			var (
				evens []int
				odds  []int
			)

			evenSink := &confluence.UnarySink[int]{}
			evenSink.Sink = func(_ context.Context, v int) error {
				evens = append(evens, v)
				return nil
			}
			plumber.SetSink(pipe, "even", evenSink)

			oddSink := &confluence.UnarySink[int]{}
			oddSink.Sink = func(_ context.Context, v int) error {
				odds = append(odds, v)
				return nil
			}
			plumber.SetSink(pipe, "odd", oddSink)

			sw := &confluence.Switch[int]{}
			sw.Switch = func(_ context.Context, v int) (address.Address, bool, error) {
				if v%2 == 0 {
					return "even", true, nil
				}
				return "odd", true, nil
			}
			plumber.SetSegment(pipe, "switch", sw)

			plumber.MultiRouter[int]{
				SourceTargets: []address.Address{"emitterOne", "emitterTwo"},
				SinkTargets:   []address.Address{"t1", "t2"},
				Stitch:        plumber.StitchUnary,
			}.MustRoute(pipe)

			plumber.MultiRouter[int]{
				SourceTargets: []address.Address{"t1", "t2"},
				SinkTargets:   []address.Address{"switch"},
				Stitch:        plumber.StitchUnary,
			}.MustRoute(pipe)

			plumber.MultiRouter[int]{
				SourceTargets: []address.Address{"switch"},
				SinkTargets:   []address.Address{"even", "odd"},
				Stitch:        plumber.StitchWeave,
			}.MustRoute(pipe)

			ctx, cancel := signal.Isolated()
			defer cancel()
			pipe.Flow(ctx, confluence.CloseOutputInletsOnExit())

			Expect(ctx.Wait()).To(MatchError("done counting"))

			Expect(len(evens) + len(odds)).To(Equal(9))
		})
	})
})
