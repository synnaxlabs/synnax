// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package op_test

import (
	"testing"

	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/telem/op"
)

var sizes = []int{100, 1000, 10000, 100000, 1000000}

func BenchmarkGreaterThanF32(b *testing.B) {
	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			a := make([]float32, size)
			bData := make([]float32, size)
			output := make([]uint8, size)
			for i := range a {
				a[i] = float32(i)
				bData[i] = float32(i + 1)
			}
			aSeries := telem.NewSeriesV(a...)
			bSeries := telem.NewSeriesV(bData...)
			outSeries := telem.NewSeriesV(output...)
			b.ReportAllocs()
			b.ResetTimer()
			b.SetBytes(int64(size * 4))
			for i := 0; i < b.N; i++ {
				op.GreaterThanF32(aSeries, bSeries, &outSeries)
			}
		})
	}
}

func BenchmarkGreaterThanF64(b *testing.B) {
	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			a := make([]float64, size)
			bData := make([]float64, size)
			output := make([]uint8, size)
			for i := range a {
				a[i] = float64(i)
				bData[i] = float64(i + 1)
			}
			aSeries := telem.NewSeriesV(a...)
			bSeries := telem.NewSeriesV(bData...)
			outSeries := telem.NewSeriesV(output...)
			b.ReportAllocs()
			b.ResetTimer()
			b.SetBytes(int64(size * 8))
			for i := 0; i < b.N; i++ {
				op.GreaterThanF64(aSeries, bSeries, &outSeries)
			}
		})
	}
}

func BenchmarkGreaterThanI32(b *testing.B) {
	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			a := make([]int32, size)
			bData := make([]int32, size)
			output := make([]uint8, size)
			for i := range a {
				a[i] = int32(i)
				bData[i] = int32(i + 1)
			}
			aSeries := telem.NewSeriesV(a...)
			bSeries := telem.NewSeriesV(bData...)
			outSeries := telem.NewSeriesV(output...)
			b.ReportAllocs()
			b.ResetTimer()
			b.SetBytes(int64(size * 4))
			for i := 0; i < b.N; i++ {
				op.GreaterThanI32(aSeries, bSeries, &outSeries)
			}
		})
	}
}

func BenchmarkGreaterThanI64(b *testing.B) {
	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			a := make([]int64, size)
			bData := make([]int64, size)
			output := make([]uint8, size)
			for i := range a {
				a[i] = int64(i)
				bData[i] = int64(i + 1)
			}
			aSeries := telem.NewSeriesV(a...)
			bSeries := telem.NewSeriesV(bData...)
			outSeries := telem.NewSeriesV(output...)
			b.ReportAllocs()
			b.ResetTimer()
			b.SetBytes(int64(size * 8))
			for i := 0; i < b.N; i++ {
				op.GreaterThanI64(aSeries, bSeries, &outSeries)
			}
		})
	}
}

func BenchmarkAddF32(b *testing.B) {
	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			a := make([]float32, size)
			bData := make([]float32, size)
			output := make([]float32, size)
			for i := range a {
				a[i] = float32(i)
				bData[i] = float32(i + 1)
			}
			aSeries := telem.NewSeriesV(a...)
			bSeries := telem.NewSeriesV(bData...)
			outSeries := telem.NewSeriesV(output...)
			b.ReportAllocs()
			b.ResetTimer()
			b.SetBytes(int64(size * 4))
			for i := 0; i < b.N; i++ {
				op.AddF32(aSeries, bSeries, &outSeries)
			}
		})
	}
}

func BenchmarkAddF64(b *testing.B) {
	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			a := make([]float64, size)
			bData := make([]float64, size)
			output := make([]float64, size)
			for i := range a {
				a[i] = float64(i)
				bData[i] = float64(i + 1)
			}
			aSeries := telem.NewSeriesV(a...)
			bSeries := telem.NewSeriesV(bData...)
			outSeries := telem.NewSeriesV(output...)
			b.ReportAllocs()
			b.ResetTimer()
			b.SetBytes(int64(size * 8))
			for i := 0; i < b.N; i++ {
				op.AddF64(aSeries, bSeries, &outSeries)
			}
		})
	}
}

func BenchmarkMultiplyF32(b *testing.B) {
	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			a := make([]float32, size)
			bData := make([]float32, size)
			output := make([]float32, size)
			for i := range a {
				a[i] = float32(i)
				bData[i] = float32(i + 1)
			}
			aSeries := telem.NewSeriesV(a...)
			bSeries := telem.NewSeriesV(bData...)
			outSeries := telem.NewSeriesV(output...)
			b.ReportAllocs()
			b.ResetTimer()
			b.SetBytes(int64(size * 4))
			for i := 0; i < b.N; i++ {
				op.MultiplyF32(aSeries, bSeries, &outSeries)
			}
		})
	}
}

func BenchmarkMultiplyF64(b *testing.B) {
	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			a := make([]float64, size)
			bData := make([]float64, size)
			output := make([]float64, size)
			for i := range a {
				a[i] = float64(i)
				bData[i] = float64(i + 1)
			}
			aSeries := telem.NewSeriesV(a...)
			bSeries := telem.NewSeriesV(bData...)
			outSeries := telem.NewSeriesV(output...)
			b.ReportAllocs()
			b.ResetTimer()
			b.SetBytes(int64(size * 8))
			for i := 0; i < b.N; i++ {
				op.MultiplyF64(aSeries, bSeries, &outSeries)
			}
		})
	}
}

func BenchmarkDivideF32(b *testing.B) {
	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			a := make([]float32, size)
			bData := make([]float32, size)
			output := make([]float32, size)
			for i := range a {
				a[i] = float32(i + 1)
				bData[i] = float32(i + 2)
			}
			aSeries := telem.NewSeriesV(a...)
			bSeries := telem.NewSeriesV(bData...)
			outSeries := telem.NewSeriesV(output...)
			b.ReportAllocs()
			b.ResetTimer()
			b.SetBytes(int64(size * 4))
			for i := 0; i < b.N; i++ {
				op.DivideF32(aSeries, bSeries, &outSeries)
			}
		})
	}
}

func BenchmarkDivideF64(b *testing.B) {
	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			a := make([]float64, size)
			bData := make([]float64, size)
			output := make([]float64, size)
			for i := range a {
				a[i] = float64(i + 1)
				bData[i] = float64(i + 2)
			}
			aSeries := telem.NewSeriesV(a...)
			bSeries := telem.NewSeriesV(bData...)
			outSeries := telem.NewSeriesV(output...)
			b.ReportAllocs()
			b.ResetTimer()
			b.SetBytes(int64(size * 8))
			for i := 0; i < b.N; i++ {
				op.DivideF64(aSeries, bSeries, &outSeries)
			}
		})
	}
}
