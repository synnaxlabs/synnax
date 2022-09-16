package stream

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/telem"
)

type Sample struct {
	ChannelKey channel.Key
	Stamp      telem.TimeStamp
	Value      []byte
}
