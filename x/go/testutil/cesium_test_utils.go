package testutil

import (
	"github.com/synnaxlabs/cesium"
)

var k cesium.ChannelKey = 1

func GenerateCesiumChannelKey() (key cesium.ChannelKey) {
	key = k
	k++
	return
}
