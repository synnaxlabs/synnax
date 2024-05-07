package testutil

import (
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/telem"
	"math/rand"
)

func GenerateFrameAndChannels(numIndexChannels int, numDataChannels int, samplesPerChannel int) (cesium.Frame, []cesium.Channel, []cesium.ChannelKey) {
	var (
		channels      = make([]cesium.Channel, numIndexChannels+numDataChannels)
		channelKeys   = make([]cesium.ChannelKey, numIndexChannels+numDataChannels)
		series        = make([]telem.Series, numIndexChannels+numDataChannels)
		timestampData = make([]telem.TimeStamp, samplesPerChannel)
		data          = make([]int64, samplesPerChannel)
	)

	for i := 1; i <= (numIndexChannels + numDataChannels); i++ {
		channelKeys[i-1] = cesium.ChannelKey(i)
		if i <= numIndexChannels {
			ch := cesium.Channel{Key: cesium.ChannelKey(i), IsIndex: true, DataType: telem.TimeStampT}
			channels[i-1] = ch

			for j := 0; j < samplesPerChannel; j++ {
				timestampData[j] = telem.TimeStamp(j)
			}
			series[i-1] = telem.NewSeries[telem.TimeStamp](timestampData)
		} else {
			correspondingIndexChannel := cesium.ChannelKey(i%numIndexChannels + 1)
			ch := cesium.Channel{Key: cesium.ChannelKey(i), Index: correspondingIndexChannel, DataType: telem.Int64T}
			channels[i-1] = ch

			for j := 0; j < samplesPerChannel; j++ {
				data[j] = rand.Int63()
			}
			series[i-1] = telem.NewSeries[int64](data)
		}
	}

	return cesium.NewFrame(channelKeys, series), channels, channelKeys
}
