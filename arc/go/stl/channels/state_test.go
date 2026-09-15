// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channels_test

import (
	"context"
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// bound is a channel state reached through its WASM host bindings, the only path
// a program reads or writes it through.
type bound struct {
	rt *testutil.Runtime
	ps *channels.ProgramState
}

func newBound(ctx context.Context, digests ...channels.Digest) *bound {
	GinkgoHelper()
	rt := testutil.NewRuntime(ctx)
	ps := channels.NewProgramState(digests)
	MustSucceed(channels.NewHost(ctx, rt.Underlying(), ps, strings.NewProgramState()))
	rt.Passthrough(ctx, "channels")
	DeferCleanup(rt.Close)
	return &bound{rt: rt, ps: ps}
}

// readF32 returns the latest f32 on key. ok is false when the read missed.
func (b *bound) readF32(ctx context.Context, key uint32) (float32, bool) {
	v := b.rt.Call(ctx, "channels", "read_f32", testutil.U32(key))[0]
	_, missed := b.ps.TakeMissingRead()
	return testutil.AsF32(v), !missed
}

// readF64 returns the latest f64 on key. ok is false when the read missed.
func (b *bound) readF64(ctx context.Context, key uint32) (float64, bool) {
	v := b.rt.Call(ctx, "channels", "read_f64", testutil.U32(key))[0]
	_, missed := b.ps.TakeMissingRead()
	return testutil.AsF64(v), !missed
}

// readI32 returns the latest i32 on key. ok is false when the read missed.
func (b *bound) readI32(ctx context.Context, key uint32) (int32, bool) {
	v := b.rt.Call(ctx, "channels", "read_i32", testutil.U32(key))[0]
	_, missed := b.ps.TakeMissingRead()
	return int32(testutil.AsU32(v)), !missed
}

func (b *bound) writeF32(ctx context.Context, key uint32, v float32) {
	b.rt.CallVoid(ctx, "channels", "write_f32", testutil.U32(key), testutil.F32(v))
}

func (b *bound) writeF64(ctx context.Context, key uint32, v float64) {
	b.rt.CallVoid(ctx, "channels", "write_f64", testutil.U32(key), testutil.F64(v))
}

func (b *bound) writeI32(ctx context.Context, key uint32, v int32) {
	b.rt.CallVoid(ctx, "channels", "write_i32", testutil.U32(key), testutil.I32(v))
}

func (b *bound) flush() (telem.Frame[uint32], bool) {
	return b.ps.Flush(telem.Frame[uint32]{})
}

var _ = Describe("ProgramState", func() {
	var b *bound

	BeforeEach(func(ctx SpecContext) {
		b = newBound(ctx,
			channels.Digest{Key: 1, DataType: telem.Float32T, Index: 2},
			channels.Digest{Key: 3, DataType: telem.Int32T},
			channels.Digest{Key: 5, DataType: telem.Float64T, Index: 6},
		)
	})

	Describe("NewProgramState", func() {
		It(
			"Should stamp writes through the digests' index channels",
			func(ctx SpecContext) {
				b2 := newBound(ctx,
					channels.Digest{Key: 10, Index: 11},
					channels.Digest{Key: 20, Index: 21},
				)
				b2.writeF32(ctx, 10, 1.0)
				fr, changed := b2.flush()
				Expect(changed).To(BeTrue())
				Expect(fr.Get(11).Series).To(HaveLen(1))
			},
		)

		It(
			"Should read every channel as missing with no digests",
			func(ctx SpecContext) {
				b2 := newBound(ctx)
				_, ok := b2.readF32(ctx, 1)
				Expect(ok).To(BeFalse())
			},
		)

		It("Should ignore a zero index in the digests", func(ctx SpecContext) {
			b2 := newBound(ctx, channels.Digest{Key: 10, Index: 0})
			b2.writeI32(ctx, 10, 42)
			fr, changed := b2.flush()
			Expect(changed).To(BeTrue())
			Expect(fr.Get(10).Series).To(HaveLen(1))
			Expect(fr.Get(0).Series).To(BeEmpty())
		})
	})

	Describe("Ingest", func() {
		It("Should expose the last ingested sample to reads", func(ctx SpecContext) {
			b.ps.Ingest(
				telem.UnaryFrame[uint32](1, telem.NewSeriesV[float32](1.5, 2.5)),
			)
			Expect(MustBeOk(b.readF32(ctx, 1))).To(Equal(float32(2.5)))
		})

		It("Should handle multiple channels in a single frame", func(ctx SpecContext) {
			fr := telem.Frame[uint32]{}
			fr = fr.Append(1, telem.NewSeriesV[float32](10.0))
			fr = fr.Append(3, telem.NewSeriesV[int32](42))
			b.ps.Ingest(fr)
			Expect(MustBeOk(b.readF32(ctx, 1))).To(Equal(float32(10.0)))
			Expect(MustBeOk(b.readI32(ctx, 3))).To(Equal(int32(42)))
		})

		It("Should accept channels not in the digests", func(ctx SpecContext) {
			b.ps.Ingest(telem.UnaryFrame[uint32](999, telem.NewSeriesV(1.0)))
			Expect(MustBeOk(b.readF64(ctx, 999))).To(Equal(1.0))
		})

		It("Should handle an empty frame without panicking", func() {
			Expect(func() { b.ps.Ingest(telem.Frame[uint32]{}) }).ToNot(Panic())
		})

		It("Should keep boundary float values", func(ctx SpecContext) {
			b.ps.Ingest(telem.UnaryFrame[uint32](5, telem.NewSeriesV(math.MaxFloat64)))
			Expect(MustBeOk(b.readF64(ctx, 5))).To(Equal(math.MaxFloat64))
			b.ps.Ingest(telem.UnaryFrame[uint32](
				5,
				telem.NewSeriesV(
					math.SmallestNonzeroFloat64,
					math.Inf(1),
					math.Inf(-1),
				),
			))
			v, ok := b.readF64(ctx, 5)
			Expect(ok).To(BeTrue())
			Expect(math.IsInf(v, -1)).To(BeTrue())
		})

		It("Should keep NaN values", func(ctx SpecContext) {
			b.ps.Ingest(telem.UnaryFrame[uint32](5, telem.NewSeriesV(math.NaN())))
			v, ok := b.readF64(ctx, 5)
			Expect(ok).To(BeTrue())
			Expect(math.IsNaN(v)).To(BeTrue())
		})

		It("Should keep max and min integer values", func(ctx SpecContext) {
			b.ps.Ingest(
				telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](math.MaxInt32)),
			)
			Expect(MustBeOk(b.readI32(ctx, 3))).To(Equal(int32(math.MaxInt32)))
			b.ps.Ingest(
				telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](math.MinInt32)),
			)
			Expect(MustBeOk(b.readI32(ctx, 3))).To(Equal(int32(math.MinInt32)))
		})

		It("Should treat an empty series as no value", func(ctx SpecContext) {
			b.ps.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32]()))
			_, ok := b.readI32(ctx, 3)
			Expect(ok).To(BeFalse())
		})

		It(
			"Should read the latest series after many ingestions",
			func(ctx SpecContext) {
				for i := range 100 {
					b.ps.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV(int32(i))))
				}
				Expect(MustBeOk(b.readI32(ctx, 3))).To(Equal(int32(99)))
			},
		)

		It("Should read an unknown channel as missing", func(ctx SpecContext) {
			_, ok := b.readI32(ctx, 999)
			Expect(ok).To(BeFalse())
			_, ok = b.readI32(ctx, 0)
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Writes", func() {
		It("Should buffer a write until it is flushed", func(ctx SpecContext) {
			b.writeI32(ctx, 3, 100)
			fr, changed := b.flush()
			Expect(changed).To(BeTrue())
			Expect(fr.Get(3).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesV[int32](100)),
			)
		})

		It("Should stamp the index channel of an indexed write", func(ctx SpecContext) {
			b.writeF32(ctx, 1, 5.0)
			fr, changed := b.flush()
			Expect(changed).To(BeTrue())
			Expect(fr.Get(1).Series).To(HaveLen(1))
			Expect(fr.Get(2).Series).To(HaveLen(1))
			Expect(fr.Get(2).Series[0].DataType).To(Equal(telem.TimestampT))
		})

		It(
			"Should not stamp an index for a channel without one",
			func(ctx SpecContext) {
				b.writeI32(ctx, 3, 50)
				fr, _ := b.flush()
				Expect(fr.Get(3).Series).To(HaveLen(1))
				Expect(fr.Get(0).Series).To(BeEmpty())
			},
		)

		It("Should accumulate writes within a cycle", func(ctx SpecContext) {
			b.writeI32(ctx, 3, 1)
			b.writeI32(ctx, 3, 2)
			fr, _ := b.flush()
			Expect(fr.Get(3).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesV[int32](1, 2)),
			)
		})

		It(
			"Should accept writes to channels not in the digests",
			func(ctx SpecContext) {
				b.writeF64(ctx, 888, 3.14)
				fr, changed := b.flush()
				Expect(changed).To(BeTrue())
				Expect(fr.Get(888).Series[0]).To(
					telem.MatchSeries(telem.NewSeriesV(3.14)),
				)
			},
		)
	})

	Describe("Flush", func() {
		It("Should return false when no writes are buffered", func() {
			_, changed := b.flush()
			Expect(changed).To(BeFalse())
		})

		It("Should clear the write buffer", func(ctx SpecContext) {
			b.writeF32(ctx, 1, 9.9)
			fr, changed := b.flush()
			Expect(changed).To(BeTrue())
			Expect(fr.Get(1).Series).To(HaveLen(1))
			_, changed = b.flush()
			Expect(changed).To(BeFalse())
		})

		It("Should detach flushed data from later writes", func(ctx SpecContext) {
			b.writeF32(ctx, 1, 1.0)
			first, _ := b.flush()
			b.writeF32(ctx, 1, 9.0)
			_, _ = b.flush()
			Expect(first.Get(1).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesV[float32](1.0)),
			)
		})

		It("Should append to an existing frame", func(ctx SpecContext) {
			existing := telem.UnaryFrame[uint32](100, telem.NewSeriesV[int32](1))
			b.writeI32(ctx, 3, 2)
			fr, changed := b.ps.Flush(existing)
			Expect(changed).To(BeTrue())
			Expect(fr.Get(100).Series).To(HaveLen(1))
			Expect(fr.Get(3).Series).To(HaveLen(1))
		})

		It("Should flush writes for multiple channels", func(ctx SpecContext) {
			b.writeF32(ctx, 1, 1.0)
			b.writeI32(ctx, 3, 2)
			fr, changed := b.flush()
			Expect(changed).To(BeTrue())
			Expect(fr.Get(1).Series).To(HaveLen(1))
			Expect(fr.Get(3).Series).To(HaveLen(1))
		})
	})

	Describe("ClearReads", func() {
		It("Should keep the latest value of each channel", func(ctx SpecContext) {
			for i := range int32(3) {
				b.ps.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV(i+1)))
			}
			b.ps.ClearReads()
			Expect(MustBeOk(b.readI32(ctx, 3))).To(Equal(int32(3)))
		})

		It(
			"Should be a no-op for a channel with a single series",
			func(ctx SpecContext) {
				b.ps.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](1)))
				b.ps.ClearReads()
				Expect(MustBeOk(b.readI32(ctx, 3))).To(Equal(int32(1)))
			},
		)

		It("Should be a no-op for empty state", func() {
			Expect(func() { b.ps.ClearReads() }).ToNot(Panic())
		})

		It("Should handle multiple channels independently", func(ctx SpecContext) {
			b.ps.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](1)))
			b.ps.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](2)))
			fr := telem.Frame[uint32]{}
			fr = fr.Append(1, telem.NewSeriesV[float32](10.0))
			fr = fr.Append(2, telem.NewSeriesSecondsTSV(100))
			b.ps.Ingest(fr)
			b.ps.ClearReads()
			Expect(MustBeOk(b.readI32(ctx, 3))).To(Equal(int32(2)))
			Expect(MustBeOk(b.readF32(ctx, 1))).To(Equal(float32(10.0)))
		})

		It("Should accept new ingestions after a clear", func(ctx SpecContext) {
			b.ps.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](1)))
			b.ps.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](2)))
			b.ps.ClearReads()
			b.ps.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](3)))
			Expect(MustBeOk(b.readI32(ctx, 3))).To(Equal(int32(3)))
		})

		It(
			"Should keep the latest value past the reallocation threshold",
			func(ctx SpecContext) {
				for i := range 100 {
					b.ps.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV(int32(i))))
				}
				b.ps.ClearReads()
				Expect(MustBeOk(b.readI32(ctx, 3))).To(Equal(int32(99)))
			},
		)
	})

	Describe("Ingest then Flush roundtrip", func() {
		It("Should keep reads and writes apart", func(ctx SpecContext) {
			b.ps.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](10)))
			b.writeI32(ctx, 3, 20)
			Expect(MustBeOk(b.readI32(ctx, 3))).To(Equal(int32(10)))
			fr, _ := b.flush()
			Expect(fr.Get(3).Series[0].ValueAt[int32](0)).To(Equal(int32(20)))
		})
	})
})
