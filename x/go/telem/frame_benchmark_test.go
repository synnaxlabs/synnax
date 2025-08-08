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
	"fmt"
	"io"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/telem"
)

func createFrame() telem.Frame[int] {
	width := 12
	height := 10000
	keys := make([]int, width)
	series := make([]telem.Series, width)
	for i := range width {
		keys[i] = i
	}
	uint8Data := make([]uint8, height)
	uint16Data := make([]uint16, height)
	uint32Data := make([]uint32, height)
	uint64Data := make([]uint64, height)
	int8Data := make([]int8, height)
	int16Data := make([]int16, height)
	int32Data := make([]int32, height)
	int64Data := make([]int64, height)
	float32Data := make([]float32, height)
	float64Data := make([]float64, height)
	timeStampData := make([]telem.TimeStamp, height)
	uuidData := make(uuid.UUIDs, height)
	for i := range height {
		uint8Data[i] = uint8(i)
		uint16Data[i] = uint16(i)
		uint32Data[i] = uint32(i)
		uint64Data[i] = uint64(i)
		int8Data[i] = int8(i)
		int16Data[i] = int16(i)
		int32Data[i] = int32(i)
		int64Data[i] = int64(i)
		float32Data[i] = float32(i)
		float64Data[i] = float64(i)
		timeStampData[i] = telem.TimeStamp(i)
		uuidData[i] = uuid.New()
	}
	series[0] = telem.NewSeries(uint8Data)
	series[1] = telem.NewSeries(uint16Data)
	series[2] = telem.NewSeries(uint32Data)
	series[3] = telem.NewSeries(uint64Data)
	series[4] = telem.NewSeries(int8Data)
	series[5] = telem.NewSeries(int16Data)
	series[6] = telem.NewSeries(int32Data)
	series[7] = telem.NewSeries(int64Data)
	series[8] = telem.NewSeries(float32Data)
	series[9] = telem.NewSeries(float64Data)
	series[10] = telem.NewSeries(timeStampData)
	series[11] = telem.NewSeriesUUIDs(uuidData)
	return telem.MultiFrame(keys, series)
}

func seriesToCSVStrings(s telem.Series) []string {
	switch s.DataType {
	case telem.Int8T:
		return lo.Map(telem.UnmarshalSeries[int8](s), func(v int8, _ int) string {
			return strconv.FormatInt(int64(v), 10)
		})
	case telem.Int16T:
		return lo.Map(telem.UnmarshalSeries[int16](s), func(v int16, _ int) string {
			return strconv.FormatInt(int64(v), 10)
		})
	case telem.Int32T:
		return lo.Map(telem.UnmarshalSeries[int32](s), func(v int32, _ int) string {
			return strconv.FormatInt(int64(v), 10)
		})
	case telem.Int64T:
		return lo.Map(telem.UnmarshalSeries[int64](s), func(v int64, _ int) string {
			return strconv.FormatInt(v, 10)
		})
	case telem.Uint8T:
		return lo.Map(telem.UnmarshalSeries[uint8](s), func(v uint8, _ int) string {
			return strconv.FormatUint(uint64(v), 10)
		})
	case telem.Uint16T:
		return lo.Map(telem.UnmarshalSeries[uint16](s), func(v uint16, _ int) string {
			return strconv.FormatUint(uint64(v), 10)
		})
	case telem.Uint32T:
		return lo.Map(telem.UnmarshalSeries[uint32](s), func(v uint32, _ int) string {
			return strconv.FormatUint(uint64(v), 10)
		})
	case telem.Uint64T:
		return lo.Map(telem.UnmarshalSeries[uint64](s), func(v uint64, _ int) string {
			return strconv.FormatUint(v, 10)
		})
	case telem.Float32T:
		return lo.Map(telem.UnmarshalSeries[float32](s), func(v float32, _ int) string {
			return strconv.FormatFloat(float64(v), 'f', -1, 32)
		})
	case telem.Float64T:
		return lo.Map(telem.UnmarshalSeries[float64](s), func(v float64, _ int) string {
			return fmt.Sprintf("%v", v)
		})
	case telem.TimeStampT:
		return lo.Map(telem.UnmarshalSeries[telem.TimeStamp](s), func(v telem.TimeStamp, _ int) string {
			return fmt.Sprintf("%v", int64(v))
		})
	case telem.UUIDT:
		return lo.Map(telem.UnmarshalUUIDs(s.Data), func(v uuid.UUID, _ int) string {
			return v.String()
		})
	default:
		panic(fmt.Sprintf("unsupported data type: %v", s.DataType))
	}
}

func BenchmarkFrameMarshalCSV(b *testing.B) {
	fr := createFrame()
	for b.Loop() {
		var rowCount int64
		for s := range fr.Series() {
			if s.Len() > rowCount {
				rowCount = s.Len()
			}
		}
		records := make([][]string, rowCount)
		for i := range rowCount {
			records[i] = make([]string, fr.Count())
		}
		for col, s := range fr.SeriesI() {
			for row, entry := range seriesToCSVStrings(s) {
				records[row][col] = entry
			}
		}
	}
}

func BenchmarkFrameWriteCSV(b *testing.B) {
	fr := createFrame()
	for b.Loop() {
		if err := fr.WriteCSV(io.Discard); err != nil {
			b.Fatalf("failed to marshal CSV: %v", err)
		}
	}
}
