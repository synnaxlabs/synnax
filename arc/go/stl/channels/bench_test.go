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

const writesPerCycle = 8

func benchmarkFlush(b *testing.B, bodyWrites bool, stamp func(cycle, j int) int64) {
	s := channels.NewProgramState([]channels.Digest{{Key: 1, Index: 2}})
	b.ReportAllocs()
	for cycle := 0; b.Loop(); cycle++ {
		base := int64(cycle+1) * 1000
		for j := range writesPerCycle {
			s.WriteChannel(
				1,
				telem.NewSeriesV(float32(j)),
				telem.NewSeriesV(telem.TimeStamp(base+stamp(cycle, j))),
			)
			if bodyWrites {
				s.WriteChannelF32(1, float32(j))
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
