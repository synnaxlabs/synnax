package testutil

import (
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/telem"
	"math/rand"
)

func GenerateFrameAndChannels(numIndexChannels, numDataChannels, numRateChannels, numDomainsPerChannel, numSamplesPerDomain int) ([]cesium.Frame, []cesium.Channel, []cesium.ChannelKey) {
	var (
		numTotalChannels = numIndexChannels + numDataChannels + numRateChannels
		channels         = make([]cesium.Channel, numTotalChannels)
		channelKeys      = make([]cesium.ChannelKey, numTotalChannels)
		frames           = make([]cesium.Frame, numDomainsPerChannel)
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

	//We're going to first resolve the index channels' data.
	var timeStamps = make([]telem.TimeStamp, numSamplesPerDomain)
	counter := 0
	for i := 0; i < numDomainsPerChannel; i++ {
		for j := 0; j < numSamplesPerDomain; j++ {
			counter += 1
			if counter == 1 {
				timeStamps[j] = 1 * telem.SecondTS
				continue
			}
			timeStamps[j] = telem.TimeStamp(1000*(float64(counter)+0.05*rand.NormFloat64())) * telem.MillisecondTS
		}

		for indexChannelKey := 1; indexChannelKey <= numIndexChannels; indexChannelKey++ {
			frames[i] = frames[i].Append(cesium.ChannelKey(indexChannelKey), telem.NewSeries[telem.TimeStamp](timeStamps))
		}
	}

	// Next, we're going to resolve the data channels and rate channels
	var data = make([]int64, numSamplesPerDomain)
	for i := 0; i < numSamplesPerDomain; i++ {
		data[i] = rand.Int63()
	}

	for i := 0; i < numDomainsPerChannel; i++ {
		for dataChannelKey := numIndexChannels + 1; dataChannelKey <= numTotalChannels; dataChannelKey++ {
			frames[i] = frames[i].Append(cesium.ChannelKey(dataChannelKey), telem.NewSeries[int64](data))
		}
	}

	return frames, channels, channelKeys
}
