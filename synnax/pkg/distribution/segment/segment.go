package segment

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/writer"
)

type (
	Segment        = core.Segment
	Iterator       = iterator.Iterator
	StreamIterator = iterator.StreamIterator
	Writer         = writer.Writer
	StreamWriter   = writer.StreamWriter
)
