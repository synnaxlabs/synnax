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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/synnaxlabs/x/telem"
)

// BenchmarkValueAt benchmarks the ValueAt function with the optimized CastSlice implementation.
func BenchmarkValueAt(b *testing.B) {
	series := telem.NewSeriesV(1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0)
	for i := 0; b.Loop(); i++ {
		_ = telem.ValueAt[float64](series, i%10)
	}
}

// BenchmarkSetValueAt benchmarks the SetValueAt function with the optimized CastSlice implementation.
func BenchmarkSetValueAt(b *testing.B) {
	series := telem.NewSeriesV(1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0)
	for i := 0; b.Loop(); i++ {
		telem.SetValueAt(series, i%10, float64(i))
	}
}

// BenchmarkValueAtLargeSlice benchmarks ValueAt on a larger series.
func BenchmarkValueAtLargeSlice(b *testing.B) {
	data := make([]float64, 10000)
	for i := range data {
		data[i] = float64(i)
	}
	series := telem.NewSeriesV(data...)
	for i := 0; b.Loop(); i++ {
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
	for i := 0; b.Loop(); i++ {
		telem.SetValueAt(series, i%10000, float64(i))
	}
}

func BenchmarkValidate(b *testing.B) {
	b.Run("FixedDensity", func(b *testing.B) {
		for _, size := range []int{10, 100, 1000, 10000} {
			b.Run(fmt.Sprintf("Float64/%d", size), func(b *testing.B) {
				data := make([]float64, size)
				for i := range data {
					data[i] = float64(i)
				}
				s := telem.NewSeriesV(data...)
				for b.Loop() {
					if err := s.Validate(); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})

	b.Run("String", func(b *testing.B) {
		for _, size := range []int{10, 100, 1000} {
			b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
				data := make([]string, size)
				for i := range data {
					data[i] = fmt.Sprintf("sample-%d", i)
				}
				s := telem.NewSeriesV(data...)
				for b.Loop() {
					if err := s.Validate(); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})

	b.Run("JSON", func(b *testing.B) {
		for _, size := range []int{10, 100, 1000} {
			b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
				data := make([][]byte, size)
				for i := range data {
					data[i], _ = json.Marshal(map[string]any{
						"key": fmt.Sprintf("value-%d", i),
						"num": i,
					})
				}
				s := telem.NewSeriesV(data...)
				s.DataType = telem.JSONT
				for b.Loop() {
					if err := s.Validate(); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})
}
