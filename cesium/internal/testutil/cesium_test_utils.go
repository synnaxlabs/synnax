// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/atomic"
	"github.com/synnaxlabs/x/telem"
	"math/rand"
)

var k = atomic.Int64Counter{}

func GenerateChannelKey() cesium.ChannelKey {
	return cesium.ChannelKey(k.Add(1))
}

func GenerateDataAndChannels(numIndexChannels, numDataChannels, numRateChannels, numSamplesPerDomain int) (telem.Series, []cesium.Channel, []cesium.ChannelKey) {
	var (
		numTotalChannels = numIndexChannels + numDataChannels + numRateChannels
		channels         = make([]cesium.Channel, numTotalChannels)
		channelKeys      = make([]cesium.ChannelKey, numTotalChannels)
	)

	for i := 1; i <= numTotalChannels; i++ {
		var ch cesium.Channel
		if i <= numIndexChannels {
			ch = cesium.Channel{Key: cesium.ChannelKey(i), IsIndex: true, DataType: telem.TimeStampT}
		} else if i <= numIndexChannels+numDataChannels {
			correspondingIndexChannel := cesium.ChannelKey(i%numIndexChannels + 1)
			ch = cesium.Channel{Key: cesium.ChannelKey(i), Index: correspondingIndexChannel, DataType: telem.Int64T}
		} else {
			ch = cesium.Channel{Key: cesium.ChannelKey(i), DataType: telem.Int64T, Rate: 1 * telem.Hz}
		}

		channelKeys[i-1] = cesium.ChannelKey(i)
		channels[i-1] = ch
	}

	var data = make([]int64, numSamplesPerDomain)
	for i := 0; i < numSamplesPerDomain; i++ {
		data[i] = rand.Int63()
	}

	return telem.NewSeries(data), channels, channelKeys
}
