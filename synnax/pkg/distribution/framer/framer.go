package framer

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/telem"
)

type (
	Frame          = core.Frame
	Iterator       = iterator.Iterator
	StreamIterator = iterator.StreamIterator
	Writer         = writer.Writer
	StreamWriter   = writer.StreamWriter
)

func NewFrame(keys channel.Keys, arrays []telem.Array) Frame {
	return core.NewFrame(keys, arrays)
}
