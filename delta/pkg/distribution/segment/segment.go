package segment

import (
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/delta/pkg/distribution/segment/iterator"
	"github.com/arya-analytics/delta/pkg/distribution/segment/writer"
)

type (
	Segment  = core.Segment
	Iterator = iterator.Iterator
	Writer   = writer.Writer
)
