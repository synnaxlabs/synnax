package stream

import (
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/x/telem"
)

type Sample struct {
	ChannelKey channel.Key
	Stamp      telem.TimeStamp
	Value      []byte
}
