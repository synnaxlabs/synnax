package cesium

import (
	"github.com/arya-analytics/cesium/internal/core"
	"github.com/arya-analytics/cesium/internal/persist"
	"github.com/arya-analytics/cesium/internal/segment"
	"github.com/arya-analytics/x/alamos"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/plumber"
	"github.com/arya-analytics/x/errutil"
	"github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/queue"
	"github.com/arya-analytics/x/signal"
	"github.com/arya-analytics/x/telem"
	"go.uber.org/zap"
	"go/types"
	"time"
)

// |||||| CONFIGURATION ||||||

const (
	// retrievePersistMaxRoutines is the maximum number of goroutines the retrieve query persist.Persist can use.
	retrievePersistMaxRoutines = persist.DefaultNumWorkers
	// retrieveDebounceFlushInterval is the interval at which retrieve debounce queue will flush if the number of
	// retrieve operations is below the threshold.
	retrieveDebounceFlushInterval = 5 * time.Millisecond
	// retrieveDebounceFlushThreshold is the number of retrieve operations that must be in the debounce queue before
	// it flushes
	retrieveDebounceFlushThreshold = 100
)

type readConfig struct {
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

func mergeRetrieveConfigDefaults(cfg *readConfig) {

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

type ResponseVariant uint8

const (
	// AckResponse is a response that indicates that an iteration request was acknowledged.
	AckResponse ResponseVariant = iota + 1
	// DataResponse is a response that indicates that an iteration request returned data.
	DataResponse
)

type IterCommand uint8

const (
	IterNext IterCommand = iota + 1
	IterPrev
	IterFirst
	IterLast
	NextSpan
	IterPrevSpan
	IterRange
	IterSeekFirst
	IterSeekLast
	IterSeekLT
	IterSeekGE
	IterValid
	IterError
	IterClose
)

type IteratorRequest struct {
	Command IterCommand
	Span    telem.TimeSpan
	Range   telem.TimeRange
	Stamp   telem.TimeStamp
}

// IteratorResponse is a response containing segments satisfying a Retrieve Query as well as any errors
// encountered during the retrieval.
type IteratorResponse struct {
	Counter  int
	Command  IterCommand
	Variant  ResponseVariant
	Ack      bool
	Err      error
	Segments []segment.Segment
}

func startReadPipeline(ctx signal.Context, cfg readConfig) (confluence.Inlet[[]retrieveOperationUnary], error) {
	mergeRetrieveConfigDefaults(&cfg)

	pipe := plumber.New()

	//// queue 'debounces' operations so that they can be flushed to disk in efficient
	//// batches.
	//plumber.SetSegment[[]retrieveOperationUnary, []retrieveOperationUnary](
	//	pipe,
	//	"queue",
	//	&queue.Debounce[retrieveOperationUnary]{Config: cfg.debounce},
	//)

	// batch groups operations into batches that optimize sequential IO.
	plumber.SetSegment[[]retrieveOperationUnary, []retrieveOperationSet](
		pipe,
		"batch",
		newRetrieveBatch(),
	)

	// persist executes batched operations on disk.
	plumber.SetSink[[]retrieveOperationSet](
		pipe,
		"persist",
		persist.New[core.FileKey, retrieveOperationSet](cfg.fs, cfg.persist),
	)

	c := errutil.NewCatch()

	c.Exec(plumber.UnaryRouter[[]retrieveOperationUnary]{
		SourceTarget: "query",
		SinkTarget:   "batch",
		Capacity:     10,
	}.MustRoute(pipe))

	//commandCounter.Exec(plumber.UnaryRouter[[]retrieveOperationUnary]{
	//	SourceTarget: "queue",
	//	SinkTarget:   "batch",
	//	Capacity:     10,
	//}.MustRoute(pipe))

	c.Exec(plumber.UnaryRouter[[]retrieveOperationSet]{
		SourceTarget: "batch",
		SinkTarget:   "persist",
		Capacity:     10,
	}.MustRoute(pipe))

	if err := c.Error(); err != nil {
		panic(c.Error())
	}

	pipe.Flow(ctx)

	seg := &plumber.Segment[[]IteratorRequest, types.Nil]{}

	return queryFactory, nil
}
