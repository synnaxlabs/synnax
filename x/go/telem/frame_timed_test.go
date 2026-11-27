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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// onClock returns s with its clock set to clock.
func onClock(s telem.Series, clock uint32) telem.Series {
	s.Clock = clock
	return s
}

var _ = Describe("KeepTimed", func() {
	var (
		stamps = onClock(telem.NewSeriesSecondsTSV(1, 2), 5)
		data   = onClock(telem.NewSeriesV[float32](1, 2), 5)
	)

	It("Should keep the time series that a kept series names as its clock", func() {
		fr := telem.MultiFrame([]uint32{5, 6}, []telem.Series{stamps, data})
		filtered := fr.KeepTimed(set.New[uint32](6), nil)
		Expect(filtered.KeysSlice()).To(Equal([]uint32{5, 6}))
		Expect(filtered.SeriesSlice()).To(Equal([]telem.Series{stamps, data}))
	})

	It("Should keep a time series that comes after the series that names it", func() {
		fr := telem.MultiFrame([]uint32{6, 5}, []telem.Series{data, stamps})
		Expect(fr.KeepTimed(set.New[uint32](6), nil).KeysSlice()).
			To(Equal([]uint32{6, 5}))
	})

	It("Should drop a time series that only an untimed series names", func() {
		fr := telem.MultiFrame([]uint32{5, 6}, []telem.Series{stamps, data})
		filtered := fr.KeepTimed(set.New[uint32](6), set.New[uint32](6))
		Expect(filtered.KeysSlice()).To(Equal([]uint32{6}))
	})

	It(
		"Should keep a time series that a timed series names beside an untimed one",
		func() {
			other := onClock(telem.NewSeriesV[int8](3, 4), 5)
			fr := telem.MultiFrame(
				[]uint32{5, 6, 7},
				[]telem.Series{stamps, data, other},
			)
			filtered := fr.KeepTimed(set.New[uint32](6, 7), set.New[uint32](6))
			Expect(filtered.KeysSlice()).To(Equal([]uint32{5, 6, 7}))
		},
	)

	It("Should drop a time series that no kept series names", func() {
		otherStamps := onClock(telem.NewSeriesSecondsTSV(3), 9)
		otherData := onClock(telem.NewSeriesV[float32](3), 9)
		fr := telem.MultiFrame(
			[]uint32{5, 6, 9, 10},
			[]telem.Series{stamps, data, otherStamps, otherData},
		)
		Expect(fr.KeepTimed(set.New[uint32](6), nil).KeysSlice()).
			To(Equal([]uint32{5, 6}))
	})

	It("Should keep a requested time series once", func() {
		fr := telem.MultiFrame([]uint32{5, 6}, []telem.Series{stamps, data})
		Expect(fr.KeepTimed(set.New[uint32](5, 6), nil).Count()).To(Equal(2))
	})

	It("Should keep no time series for a kept series with no clock", func() {
		untimed := telem.NewSeriesV[float32](1, 2)
		fr := telem.MultiFrame([]uint32{5, 6}, []telem.Series{stamps, untimed})
		Expect(fr.KeepTimed(set.New[uint32](6), nil).KeysSlice()).
			To(Equal([]uint32{6}))
	})

	It("Should filter a frame with more than 128 entries", func() {
		keys := make([]uint32, 0, 260)
		series := make([]telem.Series, 0, 260)
		for i := range uint32(130) {
			clock := 1000 + i
			keys = append(keys, i, clock)
			series = append(
				series,
				onClock(telem.NewSeriesV(float32(i)), clock),
				onClock(telem.NewSeriesSecondsTSV(telem.TimeStamp(i)), clock),
			)
		}
		filtered := telem.MultiFrame(keys, series).KeepTimed(set.New[uint32](3), nil)
		Expect(filtered.KeysSlice()).To(Equal([]uint32{3, 1003}))
	})
})
