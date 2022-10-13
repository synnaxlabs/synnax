package v2

import (
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/x/telem"
)

type Segment struct {
	Channel   channel.Key
	Alignment telem.Offset
	Density   telem.Density
	Data      []byte
}

type Channel struct {
	Key     channel.Key
	Indexed bool
	Indexes []channel.Key
}
