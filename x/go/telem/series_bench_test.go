// Copyright 2025 Synnax Labs, Inc.
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

// BenchmarkValueAt benchmarks the ValueAt function with the optimized CastSlice implementation.
func BenchmarkValueAt(b *testing.B) {
	series := telem.NewSeriesV[float64](1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = telem.ValueAt[float64](series, i%10)
	}
}

// BenchmarkSetValueAt benchmarks the SetValueAt function with the optimized CastSlice implementation.
func BenchmarkSetValueAt(b *testing.B) {
	series := telem.NewSeriesV[float64](1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		telem.SetValueAt[float64](series, i%10, float64(i))
	}
}

// BenchmarkValueAtLargeSlice benchmarks ValueAt on a larger series.
func BenchmarkValueAtLargeSlice(b *testing.B) {
	data := make([]float64, 10000)
	for i := range data {
		data[i] = float64(i)
	}
	series := telem.NewSeriesV(data...)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = telem.ValueAt[float64](series, i%10000)
	}
}

// BenchmarkSetValueAtLargeSlice benchmarks SetValueAt on a larger series.
func BenchmarkSetValueAtLargeSlice(b *testing.B) {
	data := make([]float64, 10000)
	for i := range data {
		data[i] = float64(i)
	}
	series := telem.NewSeriesV(data...)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		telem.SetValueAt[float64](series, i%10000, float64(i))
	}
}