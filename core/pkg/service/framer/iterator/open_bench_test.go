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
	"testing"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkIteratorOpen measures the open and close of an iterator over one calculated
// channel, which compiles its calculation.
func BenchmarkIteratorOpen(b *testing.B) {
	env := newBenchIterEnv(b)
	defer env.close(b)
	indexCh, dataChannels := env.createChannels(b, "open", 1)
	env.writeData(b, indexCh, dataChannels, 100)
	calc := env.createCalculation(b, "open_calc", "return open_sensor_0 * 2")
	cfg := iterator.Config{
		Keys:   []channel.Key{calc.Key(), calc.Index()},
		Bounds: telem.TimeRangeMax,
	}
	b.ReportAllocs()
	for b.Loop() {
		iter, err := env.iteratorSvc.Open(env.ctx, cfg)
		if err != nil {
			b.Fatalf("failed to open iterator: %v", err)
		}
		if err = iter.Close(); err != nil {
			b.Fatalf("failed to close iterator: %v", err)
		}
	}
}
