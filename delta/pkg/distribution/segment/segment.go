package segment

import (
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/delta/pkg/distribution/segment/iterator"
	"github.com/arya-analytics/delta/pkg/distribution/segment/writer"
)

type (
	Segment        = core.Segment
	Iterator       = iterator.Iterator
	StreamIterator = iterator.StreamIterator
	Writer         = writer.Writer
	StreamWriter   = writer.StreamWriter
)
