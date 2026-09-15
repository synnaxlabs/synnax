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
	"github.com/synnaxlabs/x/telem"
)

func benchmarkChannelStateForWrites(indexed bool) *channels.ProgramState {
	digest := channels.Digest{Key: 1}
	if indexed {
		digest.Index = 2
	}
	return channels.NewProgramState([]channels.Digest{digest})
}

func BenchmarkWriteSampleU8Indexed(b *testing.B) {
	s := benchmarkChannelStateForWrites(true)
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		channels.WriteSample(s, 1, uint8(i))
	}
}

func BenchmarkWriteSampleU8NoIndex(b *testing.B) {
	s := benchmarkChannelStateForWrites(false)
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		channels.WriteSample(s, 1, uint8(i))
	}
}

func BenchmarkWriteSampleU8SameKeyFlush(b *testing.B) {
	const writesPerCycle = 128
	s := benchmarkChannelStateForWrites(true)
	b.ReportAllocs()
	for b.Loop() {
		for j := range writesPerCycle {
			channels.WriteSample(s, 1, uint8(j))
		}
		_, _ = s.Flush(telem.Frame[uint32]{})
	}
}

func BenchmarkFlushManyKeysSingleWrite(b *testing.B) {
	const keys = 256
	digests := make([]channels.Digest, keys)
	for i := range keys {
		digests[i] = channels.Digest{Key: uint32(i + 1)}
	}
	s := channels.NewProgramState(digests)
	b.ReportAllocs()
	for b.Loop() {
		for k := range keys {
			channels.WriteSample(s, uint32(k+1), uint8(k))
		}
		_, _ = s.Flush(telem.Frame[uint32]{})
	}
}
