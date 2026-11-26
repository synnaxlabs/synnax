// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem_test

import (
	"testing"

	"github.com/synnaxlabs/x/telem"
)

// BenchmarkConvert converts a series of 1000 int16 samples to float64.
func BenchmarkConvert(b *testing.B) {
	s := telem.NewSeries(make([]int16, 1000))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := telem.Convert(s, telem.Float64T); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValueAs reads the last sample of an int16 series as float64.
func BenchmarkValueAs(b *testing.B) {
	s := telem.NewSeriesV[int16](1, 2, 3)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := telem.ValueAs[float64](s, -1); err != nil {
			b.Fatal(err)
		}
	}
}
