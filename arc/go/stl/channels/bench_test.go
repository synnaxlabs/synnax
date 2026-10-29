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
	"testing"

	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/stl/strings"
	stltest "github.com/synnaxlabs/arc/stl/testutil"
	"github.com/synnaxlabs/x/telem"
)

// benchmarkHost binds the channel host functions to s and returns a runtime that
// calls them, closing it when the benchmark ends.
func benchmarkHost(b *testing.B, s *channels.ProgramState) *stltest.Runtime {
	ctx := context.Background()
	rt := stltest.NewRuntime(ctx)
	b.Cleanup(func() {
		if err := rt.Close(ctx); err != nil {
			b.Error(err)
		}
	})
	if _, err := channels.NewHost(
		ctx,
		rt.Underlying(),
		s,
		strings.NewProgramState(),
	); err != nil {
		b.Fatal(err)
	}
	rt.Passthrough(ctx, channels.Name)
	return rt
}

const writesPerCycle = 8

func benchmarkFlush(b *testing.B, bodyWrites bool, stamp func(cycle, j int) int64) {
	ctx := context.Background()
	s := channels.NewProgramState([]channels.Digest{{Key: 1, Index: 2}})
	rt := benchmarkHost(b, s)
	write, err := newWriteNode(ctx, s, 1, telem.Float32T)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for cycle := 0; b.Loop(); cycle++ {
		base := int64(cycle+1) * 1000
		for j := range writesPerCycle {
			write(
				telem.NewSeriesV(float32(j)),
				telem.NewSeriesV(telem.TimeStamp(base+stamp(cycle, j))),
			)
			if bodyWrites {
				rt.CallVoid(
					ctx,
					channels.Name,
					"write_f32",
					stltest.U32(1),
					stltest.F32(float32(j)),
				)
			}
		}
		_, _, _ = s.Flush(telem.Frame[uint32]{}, telem.TimeStamp(base+500))
	}
}

func BenchmarkFlushFlowWrites(b *testing.B) {
	benchmarkFlush(b, false, func(_, j int) int64 { return int64(j) })
}

func BenchmarkFlushMixedWrites(b *testing.B) {
	benchmarkFlush(b, true, func(_, j int) int64 { return int64(j) })
}

func BenchmarkFlushUnorderedFlowWrites(b *testing.B) {
	benchmarkFlush(b, false, func(_, j int) int64 { return int64(writesPerCycle - j) })
}

func benchmarkWriteU8(b *testing.B, digest channels.Digest) {
	ctx := context.Background()
	s := channels.NewProgramState([]channels.Digest{digest})
	rt := benchmarkHost(b, s)
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		rt.CallVoid(
			ctx,
			channels.Name,
			"write_u8",
			stltest.U32(1),
			stltest.U32(uint32(i)),
		)
	}
}

func BenchmarkWriteU8Indexed(b *testing.B) {
	benchmarkWriteU8(b, channels.Digest{Key: 1, Index: 2})
}

func BenchmarkWriteU8NoIndex(b *testing.B) {
	benchmarkWriteU8(b, channels.Digest{Key: 1})
}

func BenchmarkWriteU8SameKeyFlush(b *testing.B) {
	const writesPerFlush = 128
	ctx := context.Background()
	s := channels.NewProgramState([]channels.Digest{{Key: 1, Index: 2}})
	rt := benchmarkHost(b, s)
	b.ReportAllocs()
	for b.Loop() {
		for j := range writesPerFlush {
			rt.CallVoid(
				ctx,
				channels.Name,
				"write_u8",
				stltest.U32(1),
				stltest.U32(uint32(j)),
			)
		}
		_, _, _ = s.Flush(telem.Frame[uint32]{}, 0)
	}
}

func BenchmarkFlushManyKeysSingleWrite(b *testing.B) {
	const keys = 256
	ctx := context.Background()
	digests := make([]channels.Digest, keys)
	for i := range keys {
		digests[i] = channels.Digest{Key: uint32(i + 1)}
	}
	s := channels.NewProgramState(digests)
	rt := benchmarkHost(b, s)
	b.ReportAllocs()
	for b.Loop() {
		for k := range uint32(keys) {
			rt.CallVoid(
				ctx,
				channels.Name,
				"write_u8",
				stltest.U32(k+1),
				stltest.U32(k),
			)
		}
		_, _, _ = s.Flush(telem.Frame[uint32]{}, 0)
	}
}
