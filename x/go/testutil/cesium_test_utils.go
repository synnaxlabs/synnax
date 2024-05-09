package testutil

import (
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/atomic"
)

var k = atomic.Int64Counter{}

func GenerateChannelKey() cesium.ChannelKey {
	return cesium.ChannelKey(k.Add(1))
}
