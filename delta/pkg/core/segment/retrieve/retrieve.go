package retrieve

import (
	"context"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/x/telem"
)

type Retrieve[V Value, F Format[V]] interface {
	WhereChannelKeys(...channel.Key) Retrieve[V, F]
	WhereTimeRange(tr telem.TimeRange) Retrieve[V, F]
	Exec(ctx context.Context) CompoundRange[V, F]
}

type Iterator[V Value, F Format[V]] interface {
	First() bool
	Next() bool
	Last() bool
	Value() CompoundRange[V, F]
}

type StreamingIterator[V Value, F Format[V]] interface {
	Values() <-chan F
}
