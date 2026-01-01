// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package plumber_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	. "github.com/synnaxlabs/x/confluence/plumber"
)

var _ = Describe("Router", func() {
	var p *Pipeline
	BeforeEach(func() { p = New() })

	Describe("UnaryRouter", func() {
		It("Should establish a channel between two addresses", func() {
			source := &confluence.Emitter[int]{}
			sink := &confluence.UnarySink[int]{}
			SetSource[int](p, "source", source)
			SetSink[int](p, "sink", sink)
			router := &UnaryRouter[int]{
				SourceTarget: "source",
				SinkTarget:   "sink",
				Capacity:     1,
			}
			Expect(router.Route(p)).To(Succeed())
			source.Out.Inlet() <- 1
			Expect(sink.In.Outlet()).To(Receive(Equal(1)))
		})
		It("Should return an error if source is not found", func() {
			router := &UnaryRouter[int]{
				SourceTarget: "source",
				SinkTarget:   "sink",
				Capacity:     1,
			}
			Expect(router.Route(p)).ToNot(Succeed())
		})
		It("Should return an error if sink is not found", func() {
			source := &confluence.Emitter[int]{}
			SetSource[int](p, "source", source)
			router := &UnaryRouter[int]{
				SourceTarget: "source",
				SinkTarget:   "sink",
				Capacity:     1,
			}
			Expect(router.Route(p)).ToNot(Succeed())
		})
	})

	Describe("MultiRouter", func() {
		Describe("StitchUnary", func() {
			It("Should wire a single channel between multiple addresses", func() {
				sourceOne := &confluence.Emitter[int]{}
				sourceTwo := &confluence.Emitter[int]{}
				sinkOne := &confluence.UnarySink[int]{}
				sinkTwo := &confluence.UnarySink[int]{}
				SetSource[int](p, "sourceOne", sourceOne)
				SetSource[int](p, "sourceTwo", sourceTwo)
				SetSink[int](p, "sinkOne", sinkOne)
				SetSink[int](p, "sinkTwo", sinkTwo)
				router := &MultiRouter[int]{
					SourceTargets: []address.Address{"sourceOne", "sourceTwo"},
					SinkTargets:   []address.Address{"sinkOne", "sinkTwo"},
					Stitch:        StitchUnary,
					Capacity:      1,
				}
				Expect(router.Route(p)).To(Succeed())
				sourceOne.Out.Inlet() <- 1
				Expect(sinkOne.In.Outlet()).To(Receive(Equal(1)))
				sourceOne.Out.Inlet() <- 1
				Expect(sinkTwo.In.Outlet()).To(Receive(Equal(1)))
			})
			It("Should close channel after both sources release the inlet", func() {
				sourceOne := &confluence.Emitter[int]{}
				sourceTwo := &confluence.Emitter[int]{}
				sinkOne := &confluence.UnarySink[int]{}
				sinkTwo := &confluence.UnarySink[int]{}
				SetSource[int](p, "sourceOne", sourceOne)
				SetSource[int](p, "sourceTwo", sourceTwo)
				SetSink[int](p, "sinkOne", sinkOne)
				SetSink[int](p, "sinkTwo", sinkTwo)
				router := &MultiRouter[int]{
					SourceTargets: []address.Address{"sourceOne", "sourceTwo"},
					SinkTargets:   []address.Address{"sinkOne", "sinkTwo"},
					Stitch:        StitchUnary,
					Capacity:      1,
				}
				Expect(router.Route(p)).To(Succeed())
				sourceOne.Out.Close()
				sourceTwo.Out.Close()
				_, ok := <-sinkOne.In.Outlet()
				Expect(ok).To(BeFalse())
			})
		})

		Describe("StitchWeave", func() {
			It("Should wire a separate channel for each address pair", func() {
				source := &confluence.Switch[int]{}
				sinkOne := &confluence.UnarySink[int]{}
				sinkTwo := &confluence.UnarySink[int]{}
				SetSource[int](p, "source", source)
				SetSink[int](p, "sinkOne", sinkOne)
				SetSink[int](p, "sinkTwo", sinkTwo)
				router := &MultiRouter[int]{
					SourceTargets: []address.Address{"source"},
					SinkTargets:   []address.Address{"sinkOne", "sinkTwo"},
					Stitch:        StitchWeave,
					Capacity:      1,
				}
				Expect(router.Route(p)).To(Succeed())
				source.Out["sinkOne"].Inlet() <- 1
				Expect(sinkOne.In.Outlet()).To(Receive(Equal(1)))
				source.Out["sinkTwo"].Inlet() <- 1
				Expect(sinkTwo.In.Outlet()).To(Receive(Equal(1)))
			})
		})

		Describe("StitchConvergent", func() {
			It("Should wire a separate channel for each sink", func() {
				sourceOne := &confluence.Switch[int]{}
				sourceTwo := &confluence.Switch[int]{}
				sinkOne := &confluence.UnarySink[int]{}
				sinkTwo := &confluence.UnarySink[int]{}
				SetSource[int](p, "sourceOne", sourceOne)
				SetSource[int](p, "sourceTwo", sourceTwo)
				SetSink[int](p, "sinkOne", sinkOne)
				SetSink[int](p, "sinkTwo", sinkTwo)
				router := &MultiRouter[int]{
					SourceTargets: []address.Address{"sourceOne", "sourceTwo"},
					SinkTargets:   []address.Address{"sinkOne", "sinkTwo"},
					Stitch:        StitchConvergent,
					Capacity:      1,
				}
				router.MustRoute(p)
				sourceOne.Out["sinkOne"].Inlet() <- 1
				Expect(sinkOne.In.Outlet()).To(Receive(Equal(1)))
				sourceOne.Out["sinkTwo"].Inlet() <- 1
				Expect(sinkTwo.In.Outlet()).To(Receive(Equal(1)))
				sourceTwo.Out["sinkOne"].Inlet() <- 1
				Expect(sinkOne.In.Outlet()).To(Receive(Equal(1)))
				sourceTwo.Out["sinkTwo"].Inlet() <- 1
				Expect(sinkTwo.In.Outlet()).To(Receive(Equal(1)))
			})
		})

	})
})
