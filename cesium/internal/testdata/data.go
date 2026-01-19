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

const (
	Index1Key     cesium.ChannelKey = 1
	Data1Uint8Key cesium.ChannelKey = 2
	Data1Int64Key cesium.ChannelKey = 3

	Index2Key      cesium.ChannelKey = 4
	Data2Uint16Key cesium.ChannelKey = 5
	Data2Int64Key  cesium.ChannelKey = 6
	LegacyRateKey  cesium.ChannelKey = 7

	VirtualKey cesium.ChannelKey = 8
)

var (
	Channels = []cesium.Channel{
		{Key: Index1Key, IsIndex: true, DataType: telem.TimeStampT},
		{Key: Data1Uint8Key, Index: Index1Key, DataType: telem.Uint8T},
		{Key: Data1Int64Key, Index: Index1Key, DataType: telem.Int64T},
		{Key: Index2Key, IsIndex: true, DataType: telem.TimeStampT},
		{Key: Data2Uint16Key, Index: Index2Key, DataType: telem.Uint16T},
		{Key: Data2Int64Key, Index: Index2Key, DataType: telem.Int64T},
		{Key: LegacyRateKey, Index: Index2Key, DataType: telem.Uint32T},
		{Key: VirtualKey, Virtual: true, DataType: telem.StringT},
	}
	Frames = []cesium.Frame{
		telem.MultiFrame[cesium.ChannelKey](
			[]cesium.ChannelKey{Index1Key, Data1Uint8Key, Data1Int64Key, Index2Key},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(0, 1, 2, 3, 5, 6, 7, 9),
				telem.NewSeriesV[uint8](10, 11, 12, 13, 15, 16, 17, 19),
				telem.NewSeriesV[int64](100, 101, 102, 103, 105, 106, 107, 109),
				telem.NewSeriesSecondsTSV(0, 1, 2, 3, 6, 7, 8, 9),
			}),
		telem.MultiFrame[cesium.ChannelKey]([]cesium.ChannelKey{Data2Uint16Key, Data2Int64Key},
			[]telem.Series{
				telem.NewSeriesV[uint16](100, 101, 102, 103, 106),
				telem.NewSeriesV[int64](1, 11, 21, 31, 61),
			}),
		telem.MultiFrame[cesium.ChannelKey](
			[]cesium.ChannelKey{Index1Key, Data1Uint8Key, Data1Int64Key},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(10, 13, 17, 18, 19),
				telem.NewSeriesV[uint8](100, 103, 107, 108, 109),
				telem.NewSeriesV[int64](100, 103, 107, 108, 109),
			}),
		telem.MultiFrame[cesium.ChannelKey](
			[]cesium.ChannelKey{Index2Key, Data2Uint16Key, Data2Int64Key},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(13, 14, 15),
				telem.NewSeriesV[uint16](130, 140, 150),
				telem.NewSeriesV[int64](13, 14, 15),
			}),
		telem.MultiFrame[cesium.ChannelKey](
			[]cesium.ChannelKey{Index2Key, Data2Int64Key},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(20, 25, 30),
				telem.NewSeriesV[int64](2, 2, 3),
			}),
	}
)
