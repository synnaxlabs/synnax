// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package join_test

import (
	"testing"

	"github.com/synnaxlabs/arc/stl/join"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkJoin joins one second of a primary input at rate with a second input at
// other per operation, one frame of each input per sample of the primary, as a live
// read does. It reports allocations, which stay at zero once the buffers grow.
func BenchmarkJoin(b *testing.B) {
	for _, c := range []struct {
		name        string
		rate, other telem.Rate
	}{
		{
			name:  "clocks=1/rate=1kHz",
			rate:  telem.Kilohertz,
			other: telem.Kilohertz,
		},
		{
			name:  "clocks=2/rate=1kHz+1Hz",
			rate:  telem.Kilohertz,
			other: telem.Kilohertz / 1000,
		},
		{
			name:  "clocks=1/rate=10kHz",
			rate:  10 * telem.Kilohertz,
			other: 10 * telem.Kilohertz,
		},
	} {
		b.Run(c.name, func(b *testing.B) {
			j, err := join.New(join.Config{
				DataTypes: []telem.DataType{telem.Float64T, telem.Float64T},
			})
			if err != nil {
				b.Fatal(err)
			}
			var (
				samples = int(c.rate)
				step    = c.rate.Period()
				every   = int(c.rate / c.other)
				value   = telem.NewSeriesV[float64](1)
				stamp   = telem.NewSeriesV[telem.TimeStamp](0)
				out     join.Output
				now     telem.TimeStamp
				k       telem.Alignment
			)
			b.ReportAllocs()
			for b.Loop() {
				for i := range samples {
					now += telem.TimeStamp(step)
					k++
					telem.ByteOrder.PutUint64(stamp.Data, uint64(now))
					stamp.Alignment = k
					if i%every == 0 {
						if err = j.Push(1, value, stamp); err != nil {
							b.Fatal(err)
						}
					}
					if err = j.Push(0, value, stamp); err != nil {
						b.Fatal(err)
					}
					j.Next(now, &out)
				}
				out.Reset()
			}
		})
	}
}
