// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator_test

import (
	"fmt"
	"testing"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkIteratorReduced opens a min-max limit read of a calculated channel and reads
// it to the end, so each iteration runs the calculation and the reducer.
func BenchmarkIteratorReduced(b *testing.B) {
	env := newBenchIterEnv(b)
	defer env.close(b)
	indexCh, data := env.createChannels(b, "reduce", 1)
	env.writeData(b, indexCh, data, 100_000)
	calc := env.createCalculation(b, "reduce_calc", "return reduce_sensor_0 * 2")
	for _, limit := range []uint32{100, 10_000} {
		b.Run(fmt.Sprintf("limit=%d", limit), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				iter, err := env.iteratorSvc.Open(env.ctx, iterator.Config{
					Keys:   []channel.Key{calc.Key(), calc.Index()},
					Bounds: telem.TimeRangeMax,
					Reduction: telem.Reduction{Variant: telem.LimitReduction{
						Aggregation: telem.AggregationMinMax,
						PointLimit:  limit,
					}},
				})
				if err != nil {
					b.Fatal(err)
				}
				iter.SeekFirst()
				for iter.Next(iterator.AutoSpan) {
				}
				if err := iter.Close(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
