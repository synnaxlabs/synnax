// Copyright 2024 Synnax Labs, Inc.
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
	index1   cesium.ChannelKey = 1
	basic1   cesium.ChannelKey = 2
	basic2   cesium.ChannelKey = 3
	index2   cesium.ChannelKey = 4
	basic3   cesium.ChannelKey = 5
	basic4   cesium.ChannelKey = 6
	rate     cesium.ChannelKey = 7
	virtual  cesium.ChannelKey = 8
	Channels                   = []cesium.Channel{
		{Key: index1, IsIndex: true, DataType: telem.TimeStampT},
		{Key: basic1, Index: index1, DataType: telem.Uint8T},
		{Key: basic2, Index: index1, DataType: telem.Int64T},
		{Key: index2, IsIndex: true, DataType: telem.TimeStampT},
		{Key: basic3, Index: index2, DataType: telem.Uint16T},
		{Key: basic4, Index: index2, DataType: telem.Int64T},
		{Key: virtual, Virtual: true, DataType: telem.StringT},
		{Key: rate, Rate: 2 * telem.Hz, DataType: telem.Uint32T},
	}
	Frames = []cesium.Frame{
		cesium.NewFrame(
			[]cesium.ChannelKey{index1, basic1, basic2, index2, rate},
			[]telem.Series{
				telem.NewSecondsTSV(0, 1, 2, 3, 5, 6, 7, 9),
				telem.NewSeriesV[uint8](10, 11, 12, 13, 15, 16, 17, 19),
				telem.NewSeriesV[int64](100, 101, 102, 103, 105, 106, 107, 109),
				telem.NewSecondsTSV(0, 1, 2, 3, 6, 7, 8, 9),
				telem.NewSeriesV[uint32](0, 5, 10, 15, 20, 25, 30, 35),
			}),
		cesium.NewFrame([]cesium.ChannelKey{basic3, basic4},
			[]telem.Series{
				telem.NewSeriesV[uint16](100, 101, 102, 103, 106),
				telem.NewSeriesV[int64](1, 11, 21, 31, 61),
			}),
		cesium.NewFrame(
			[]cesium.ChannelKey{index1, basic1, basic2, rate},
			[]telem.Series{
				telem.NewSecondsTSV(10, 13, 17, 18, 19),
				telem.NewSeriesV[uint8](100, 103, 107, 108, 109),
				telem.NewSeriesV[int64](100, 103, 107, 108, 109),
				telem.NewSeriesV[uint32](1000, 1050, 1100, 1150, 1200),
			}),
		cesium.NewFrame(
			[]cesium.ChannelKey{index2, basic3, basic4},
			[]telem.Series{
				telem.NewSecondsTSV(13, 14, 15),
				telem.NewSeriesV[uint16](130, 140, 150),
				telem.NewSeriesV[int64](13, 14, 15),
			}),
		cesium.NewFrame(
			[]cesium.ChannelKey{index2, basic4},
			[]telem.Series{
				telem.NewSecondsTSV(20, 25, 30),
				telem.NewSeriesV[int64](2, 2, 3),
			}),
	}
)
