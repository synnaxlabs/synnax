package cesium

import (
	"context"
	"github.com/arya-analytics/cesium/internal/channel"
	"github.com/arya-analytics/cesium/internal/core"
	"github.com/arya-analytics/cesium/internal/persist"
	"github.com/arya-analytics/cesium/internal/segment"
	"github.com/arya-analytics/x/alamos"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/plumber"
	"github.com/arya-analytics/x/errutil"
	"github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/query"
	"github.com/arya-analytics/x/queue"
	"github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
	"time"
)

type retrieveSegment = confluence.Segment[[]retrieveOperation, []retrieveOperation]

// |||||| CONFIGURATION ||||||

const (
	// retrievePersistMaxRoutines is the maximum number of goroutines the retrieve query persist.Persist can use.
	retrievePersistMaxRoutines = persist.DefaultNumWorkers
	// retrieveDebounceFlushInterval is the interval at which retrieve debounce queue will flush if the number of
	// retrieve operations is below the threshold.
	retrieveDebounceFlushInterval = 10 * time.Millisecond
	// retrieveDebounceFlushThreshold is the number of retrieve operations that must be in the debounce queue before
	// it flushes
	retrieveDebounceFlushThreshold = 100
)

type retrieveConfig struct {
	// exp is used to track metrics for the Retrieve query. See retrieveMetrics for more.
	exp alamos.Experiment
	// fs is the file system for reading segment data from.
	fs core.FS
	// kv is the key-value store for reading segment metadata from.
	kv kv.DB
	// logger is where retrieve operations will log their progress.
	logger *zap.Logger
	// debounce sets the debounce parameters for retrieve operations.
	// this is mostly here for optimizing performance under varied conditions.
	debounce queue.DebounceConfig
	// persist used for setting the parameters for persist.Persist thar reads
	// segment data from disk.
	persist persist.Config
}

func mergeRetrieveConfigDefaults(cfg *retrieveConfig) {

	// |||||| PERSIST ||||||

	if cfg.persist.NumWorkers == 0 {
		cfg.persist.NumWorkers = retrievePersistMaxRoutines
	}
	if cfg.persist.Logger == nil {
		cfg.persist.Logger = cfg.logger
	}

	// |||||| DEBOUNCE ||||||

	if cfg.debounce.Interval == 0 {
		cfg.debounce.Interval = retrieveDebounceFlushInterval
	}
	if cfg.debounce.Threshold == 0 {
		cfg.debounce.Threshold = retrieveDebounceFlushThreshold
	}
}

// |||||| STREAM ||||||

// RetrieveResponse is a response containing segments satisfying a Retrieve Query as well as any errors
// encountered during the retrieval.
type RetrieveResponse struct {
	Segments []segment.Segment
}

// |||||| QUERY ||||||

type Retrieve struct {
	query.Query
	kve     kv.DB
	ops     confluence.Inlet[[]retrieveOperation]
	metrics retrieveMetrics
	logger  *zap.Logger
}

// WhereChannels sets the channels to retrieve data for.
// If no keys are provided, will return an ErrInvalidQuery error.
func (r Retrieve) WhereChannels(keys ...ChannelKey) Retrieve { channel.SetKeys(r, keys...); return r }

// WhereTimeRange sets the time range to retrieve data from.
func (r Retrieve) WhereTimeRange(tr TimeRange) Retrieve { setTimeRange(r, tr); return r }

// Stream streams all segments from the iterator out to the channel. Errors encountered
// during stream construction are returned immediately. Errors encountered during
// segment reads are returns as part of RetrieveResponse.
func (r Retrieve) Stream(ctx context.Context, res chan<- RetrieveResponse) (err error) {
	iter := r.Iterate()
	iter.OutTo(confluence.NewInlet(res))
	defer func() {
		err = errors.CombineErrors(iter.Close(), err)
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		if ctx.Err() != nil {
			err = ctx.Err()
		}
	}
	return err
}

func (r Retrieve) Iterate() StreamIterator {
	return newIteratorFromRetrieve(r)
}

// |||||| OPTIONS ||||||

// |||| TIME RANGE ||||

const timeRangeOptKey query.OptionKey = "tr"

func setTimeRange(q query.Query, tr TimeRange) {
	q.Set(timeRangeOptKey, tr)
}

func timeRange(q query.Query) TimeRange {
	tr, ok := q.Get(timeRangeOptKey)
	if !ok {
		return TimeRangeMax
	}
	return tr.(TimeRange)
}

// |||||| QUERY FACTORY ||||||

type retrieveFactory struct {
	kve     kv.DB
	logger  *zap.Logger
	metrics retrieveMetrics
	confluence.AbstractUnarySource[[]retrieveOperation]
	confluence.EmptyFlow
}

func (r retrieveFactory) New() Retrieve {
	return Retrieve{
		Query:   query.New(),
		ops:     r.Out,
		kve:     r.kve,
		logger:  r.logger,
		metrics: r.metrics,
	}
}

func startRetrieve(
	ctx signal.Context,
	cfg retrieveConfig,
) (query.Factory[Retrieve], error) {
	mergeRetrieveConfigDefaults(&cfg)

	pipe := plumber.New()

	// queue 'debounces' operations so that they can be flushed to disk in efficient
	// batches.
	plumber.SetSegment[[]retrieveOperation, []retrieveOperation](
		pipe,
		"queue",
		&queue.Debounce[retrieveOperation]{Config: cfg.debounce},
	)

	// batch groups operations into batches that optimize sequential IO.
	plumber.SetSegment[[]retrieveOperation, []retrieveOperation](
		pipe,
		"batch",
		newRetrieveBatch(),
	)

	// persist executes batched operations on disk.
	plumber.SetSink[[]retrieveOperation](
		pipe,
		"persist",
		persist.New[core.FileKey, retrieveOperation](cfg.fs, cfg.persist),
	)

	queryFactory := &retrieveFactory{
		kve:     cfg.kv,
		metrics: newRetrieveMetrics(cfg.exp),
		logger:  cfg.logger,
	}

	plumber.SetSource[[]retrieveOperation](pipe, "query", queryFactory)

	c := errutil.NewCatchSimple()

	c.Exec(plumber.UnaryRouter[[]retrieveOperation]{
		SourceTarget: "query",
		SinkTarget:   "queue",
		Capacity:     10,
	}.PreRoute(pipe))

	c.Exec(plumber.UnaryRouter[[]retrieveOperation]{
		SourceTarget: "queue",
		SinkTarget:   "batch",
		Capacity:     10,
	}.PreRoute(pipe))

	c.Exec(plumber.UnaryRouter[[]retrieveOperation]{
		SourceTarget: "batch",
		SinkTarget:   "persist",
		Capacity:     10,
	}.PreRoute(pipe))

	pipe.Flow(ctx)

	return queryFactory, c.Error()
}
