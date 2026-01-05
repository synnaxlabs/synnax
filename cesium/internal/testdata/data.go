// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testdata

import (
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/telem"
)

var (
	Index1     cesium.ChannelKey = 1
	Basic1     cesium.ChannelKey = 2
	Basic2     cesium.ChannelKey = 3
	Index2     cesium.ChannelKey = 4
	Basic3     cesium.ChannelKey = 5
	Basic4     cesium.ChannelKey = 6
	LegacyRate cesium.ChannelKey = 7
	Virtual    cesium.ChannelKey = 8
	Channels                     = []cesium.Channel{
		{Key: Index1, IsIndex: true, DataType: telem.TimeStampT},
		{Key: Basic1, Index: Index1, DataType: telem.Uint8T},
		{Key: Basic2, Index: Index1, DataType: telem.Int64T},
		{Key: Index2, IsIndex: true, DataType: telem.TimeStampT},
		{Key: Basic3, Index: Index2, DataType: telem.Uint16T},
		{Key: Basic4, Index: Index2, DataType: telem.Int64T},
		{Key: LegacyRate, Index: Index2, DataType: telem.Uint32T},
		{Key: Virtual, Virtual: true, DataType: telem.StringT},
	}
	Frames = []cesium.Frame{
		telem.MultiFrame[cesium.ChannelKey](
			[]cesium.ChannelKey{Index1, Basic1, Basic2, Index2},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(0, 1, 2, 3, 5, 6, 7, 9),
				telem.NewSeriesV[uint8](10, 11, 12, 13, 15, 16, 17, 19),
				telem.NewSeriesV[int64](100, 101, 102, 103, 105, 106, 107, 109),
				telem.NewSeriesSecondsTSV(0, 1, 2, 3, 6, 7, 8, 9),
			}),
		telem.MultiFrame[cesium.ChannelKey]([]cesium.ChannelKey{Basic3, Basic4},
			[]telem.Series{
				telem.NewSeriesV[uint16](100, 101, 102, 103, 106),
				telem.NewSeriesV[int64](1, 11, 21, 31, 61),
			}),
		telem.MultiFrame[cesium.ChannelKey](
			[]cesium.ChannelKey{Index1, Basic1, Basic2},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(10, 13, 17, 18, 19),
				telem.NewSeriesV[uint8](100, 103, 107, 108, 109),
				telem.NewSeriesV[int64](100, 103, 107, 108, 109),
			}),
		telem.MultiFrame[cesium.ChannelKey](
			[]cesium.ChannelKey{Index2, Basic3, Basic4},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(13, 14, 15),
				telem.NewSeriesV[uint16](130, 140, 150),
				telem.NewSeriesV[int64](13, 14, 15),
			}),
		telem.MultiFrame[cesium.ChannelKey](
			[]cesium.ChannelKey{Index2, Basic4},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(20, 25, 30),
				telem.NewSeriesV[int64](2, 2, 3),
			}),
	}
)
