package storage

import (
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/telem"
)

type (
	TS             = cesium.DB
	Channel        = cesium.Channel
	Frame          = cesium.Frame
	ChannelKey     uint16
	IteratorConfig = cesium.IteratorConfig
	WriterConfig   = cesium.WriterConfig
)

const AutoSpan = telem.TimeSpan(-1)
