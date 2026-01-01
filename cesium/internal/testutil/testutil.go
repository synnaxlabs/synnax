// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"fmt"
	"math/rand"

	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/atomic"
	"github.com/synnaxlabs/x/telem"
)

var k = atomic.Int64Counter{}

func GenerateChannelKey() cesium.ChannelKey {
	return cesium.ChannelKey(k.Add(1))
}

func GenerateDataAndChannels(
	numIndexChannels,
	numDataChannels,
	numSamplesPerDomain int,
) (telem.Series, []cesium.Channel, []cesium.ChannelKey) {
	var (
		numTotalChannels = numIndexChannels + numDataChannels
		channels         = make([]cesium.Channel, numTotalChannels)
		channelKeys      = make([]cesium.ChannelKey, numTotalChannels)
	)

	for i := 1; i <= numTotalChannels; i++ {
		var ch cesium.Channel
		if i <= numIndexChannels {
			ch = cesium.Channel{
				Name:     fmt.Sprintf("index-%d", i),
				Key:      cesium.ChannelKey(i),
				IsIndex:  true,
				DataType: telem.TimeStampT,
			}
		} else if i <= numIndexChannels+numDataChannels {
			correspondingIndexChannel := cesium.ChannelKey(i%numIndexChannels + 1)
			ch = cesium.Channel{
				Name:     fmt.Sprintf("data-%d", i),
				Key:      cesium.ChannelKey(i),
				Index:    correspondingIndexChannel,
				DataType: telem.Int64T,
			}
		}

		channelKeys[i-1] = cesium.ChannelKey(i)
		channels[i-1] = ch
	}

	var data = make([]int64, numSamplesPerDomain)
	for i := range numSamplesPerDomain {
		data[i] = rand.Int63()
	}

	return telem.NewSeries(data), channels, channelKeys
}
