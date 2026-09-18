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
	"testing"

	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/testutil"
	"github.com/synnaxlabs/x/telem"
)

// benchmarkWriter binds a channel state to its host bindings and returns it with a
// function that writes one u8 sample through write_u8.
func benchmarkWriter(
	b *testing.B,
	digests []channels.Digest,
) (*channels.ProgramState, func(key uint32, v uint8)) {
	ctx := b.Context()
	rt := testutil.NewRuntime(ctx)
	ps := channels.NewProgramState(digests)
	if _, err := channels.NewHost(
		ctx, rt.Underlying(), ps, strings.NewProgramState(),
	); err != nil {
		b.Fatal(err)
	}
	rt.Passthrough(ctx, "channels")
	b.Cleanup(func() {
		if err := rt.Close(ctx); err != nil {
			b.Error(err)
		}
	})
	return ps, func(key uint32, v uint8) {
		rt.CallVoid(
			ctx, "channels", "write_u8", testutil.U32(key), testutil.U32(uint32(v)),
		)
	}
}

func BenchmarkWriteU8Indexed(b *testing.B) {
	_, write := benchmarkWriter(b, []channels.Digest{{Key: 1, Index: 2}})
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		write(1, uint8(i))
	}
}

func BenchmarkWriteU8NoIndex(b *testing.B) {
	_, write := benchmarkWriter(b, []channels.Digest{{Key: 1}})
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		write(1, uint8(i))
	}
}

func BenchmarkWriteU8SameKeyFlush(b *testing.B) {
	const writesPerCycle = 128
	ps, write := benchmarkWriter(b, []channels.Digest{{Key: 1, Index: 2}})
	b.ReportAllocs()
	for b.Loop() {
		for j := range writesPerCycle {
			write(1, uint8(j))
		}
		_, _ = ps.Flush(telem.Frame[uint32]{})
	}
}

func BenchmarkFlushManyKeysSingleWrite(b *testing.B) {
	const keys = 256
	digests := make([]channels.Digest, keys)
	for i := range keys {
		digests[i] = channels.Digest{Key: uint32(i + 1)}
	}
	ps, write := benchmarkWriter(b, digests)
	b.ReportAllocs()
	for b.Loop() {
		for k := range keys {
			write(uint32(k+1), uint8(k))
		}
		_, _ = ps.Flush(telem.Frame[uint32]{})
	}
}
