package cesium

import (
	"context"
	"github.com/arya-analytics/cesium/internal/allocate"
	"github.com/arya-analytics/cesium/internal/channel"
	"github.com/arya-analytics/cesium/internal/core"
	"github.com/arya-analytics/cesium/internal/kv"
	"github.com/arya-analytics/cesium/internal/persist"
	"github.com/arya-analytics/cesium/internal/segment"
	"github.com/arya-analytics/x/alamos"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/plumber"
	"github.com/arya-analytics/x/errutil"
	kvx "github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/lock"
	"github.com/arya-analytics/x/query"
	"github.com/arya-analytics/x/queue"
	"github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
	"sync"
	"time"
)

type createSegment = confluence.Segment[[]createOperation, []createOperation]

// |||||| CONFIGURATION ||||||

const (
	// createPersistMaxRoutines is the maximum number of goroutines the create
	// query persist.Persist can use.
	createPersistMaxRoutines = persist.DefaultNumWorkers
	// createDebounceFlushInterval is the interval at which create debounce
	// queue will flush if the number of create operations is below the threshold.
	createDebounceFlushInterval = 10 * time.Millisecond
	// createDebounceFlushThreshold is the number of operations that must be queued
	//before create debounce queue will flush.
	createDebounceFlushThreshold = 100
	// fileCounterKey is the key for the counter that keeps track of the number of files
	// the DB has created.
	fileCounterKey = "cesium.nextFile"
)

type createConfig struct {
	// exp is used to track metrics for the Create query. See createMetrics for all the recorded values.
	exp alamos.Experiment
	// fs is the file system for writing segment data to.
	fs core.FS
	// kv is the key-value store for writing segment metadata to.
	kv kvx.DB
	// logger is where create operations will log their progress.
	logger *zap.Logger
	// allocate is used for setting the parameters for allocating a segment to  afile.
	// This setting is particularly useful in environments where the maximum number of
	// file descriptors must be limited.
	allocate allocate.Config
	// persist is used for setting the parameters for persist.Persist that writes
	// segment data to disk.
	persist persist.Config
	// debounce sets the debounce parameters for create operations.
	// this is mostly here for optimizing performance under varied conditions.
	debounce queue.DebounceConfig
}

func mergeCreateConfigDefaults(cfg *createConfig) {

	// |||||| ALLOCATION ||||||

	if cfg.allocate.MaxSize == 0 {
		cfg.allocate.MaxSize = maxFileSize
	}
	if cfg.allocate.MaxDescriptors == 0 {
		cfg.allocate.MaxDescriptors = maxFileDescriptors
	}

	// |||||| PERSIST ||||||

	if cfg.persist.NumWorkers == 0 {
		cfg.persist.NumWorkers = createPersistMaxRoutines
	}
	if cfg.persist.Logger == nil {
		cfg.persist.Logger = cfg.logger
	}

	// |||||| DEBOUNCE ||||||

	if cfg.debounce.Interval == 0 {
		cfg.debounce.Interval = createDebounceFlushInterval
	}
	if cfg.debounce.Threshold == 0 {
		cfg.debounce.Threshold = createDebounceFlushThreshold
	}
}

// |||||| STREAM ||||||

// CreateRequest is a request containing a set of segments (segment) to write to the DB.
type CreateRequest struct {
	Segments []segment.Segment
}

// CreateResponse contains any errors that occurred during the execution of the Create Query.
type CreateResponse struct {
	Error error
}

// |||||| QUERY ||||||

type Create struct {
	query.Query
	ops     confluence.Inlet[[]createOperation]
	lock    lock.KeyLock[channel.Key]
	kv      kvx.DB
	logger  *zap.Logger
	metrics createMetrics
}

// WhereChannels sets the channels to acquire a lock on for creation.
// The request stream will only accept segmentKV bound to channel with the given primary keys.
// If no keys are provided, will return an ErrInvalidQuery error.
func (c Create) WhereChannels(keys ...channel.Key) Create { channel.SetKeys(c, keys...); return c }

// Stream opens the stream.
func (c Create) Stream(ctx context.Context) (chan<- CreateRequest, <-chan CreateResponse, error) {
	keys := channel.GetKeys(c)
	requests := confluence.NewStream[CreateRequest](0)
	responses := confluence.NewStream[CreateResponse](0)

	_channels, err := kv.NewChannel(c.kv).Get(keys...)
	if err != nil {
		return nil, nil, err
	}
	if len(_channels) != len(keys) {
		return nil, nil, NotFound
	}

	query.SetContext(c, ctx)
	if !c.lock.TryLock(keys...) {
		return nil, nil, errors.New("[cesium] - lock already held")
	}

	channels := make(map[channel.Key]channel.Channel)
	for _, ch := range _channels {
		channels[ch.Key] = ch
	}

	res := confluence.AbstractUnarySource[CreateResponse]{}
	res.OutTo(responses)

	wg := &sync.WaitGroup{}

	parser := &createParser{
		ctx:       ctx,
		logger:    c.logger,
		metrics:   c.metrics,
		header:    kv.NewHeader(c.kv),
		channels:  channels,
		responses: res,
		wg:        wg,
	}

	requestDur := c.metrics.request.Stopwatch()
	requestDur.Start()
	go func() {
		defer func() {
			requestDur.Stop()
			c.lock.Unlock(keys...)
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case req, ok := <-requests.Outlet():
				if !ok {
					wg.Wait()
					responses.Close()
					return
				}
				ops, err := parser.parse(req.Segments)
				if err != nil {
					responses.Inlet() <- CreateResponse{Error: err}
				}
				wg.Add(len(ops))
				if err := signal.SendUnderContext(ctx, c.ops.Inlet(), ops); err != nil {
					return
				}
			}
		}
	}()
	return requests.Inlet(), responses.Outlet(), nil
}

// |||||| QUERY FACTORY ||||||

type createFactory struct {
	lock    lock.KeyLock[channel.Key]
	kv      kvx.DB
	logger  *zap.Logger
	header  *kv.Header
	metrics createMetrics
	confluence.AbstractUnarySource[[]createOperation]
	confluence.EmptyFlow
}

// New implements the query.Factory interface.
func (c createFactory) New() Create {
	return Create{
		Query:   query.New(),
		kv:      c.kv,
		logger:  c.logger,
		metrics: c.metrics,
		ops:     c.Out,
		lock:    c.lock,
	}
}

// |||||| START UP |||||||

func startCreate(ctx signal.Context, cfg createConfig) (query.Factory[Create], error) {

	mergeCreateConfigDefaults(&cfg)

	// a kv persisted counter that tracks the number of files that a gorpDB has created.
	// The segment allocator uses it to determine the next file to open.
	fCount, err := newFileCounter(cfg.kv, []byte(fileCounterKey))
	if err != nil {
		return nil, err
	}

	// acquires and releases the locks on channels. Acquiring locks on channels simplifies
	// the implementation of the database significantly, as we can avoid needing to
	// serialize writes to the same channel from different goroutines.
	channelLock := lock.NewLock[channel.Key]()

	pipe := plumber.New()

	// allocator allocates segments to files.
	plumber.SetSegment[[]createOperation, []createOperation](
		pipe,
		"allocator",
		newAllocator(fCount, cfg.allocate),
	)

	// queue 'debounces' operations so that they can be flushed to disk in efficient
	// batches.
	plumber.SetSegment[[]createOperation, []createOperation](
		pipe,
		"queue",
		&queue.Debounce[createOperation]{Config: cfg.debounce},
	)

	// batch groups operations into batches that are more efficient upon retrieval.
	plumber.SetSegment(pipe, "batch", newCreateBatch())

	// persist executes batched operations to disk.
	plumber.SetSink[[]createOperation](
		pipe,
		"persist",
		persist.New[core.FileKey, createOperation](cfg.fs, cfg.persist),
	)

	queryFactory := &createFactory{
		lock:    channelLock,
		kv:      cfg.kv,
		logger:  cfg.logger,
		metrics: newCreateMetrics(cfg.exp),
	}

	plumber.SetSource[[]createOperation](pipe, "query", queryFactory)

	c := errutil.NewCatchSimple()

	c.Exec(plumber.UnaryRouter[[]createOperation]{
		SourceTarget: "query",
		SinkTarget:   "allocator",
		Capacity:     1,
	}.PreRoute(pipe))

	c.Exec(plumber.UnaryRouter[[]createOperation]{
		SourceTarget: "allocator",
		SinkTarget:   "queue",
		Capacity:     1,
	}.PreRoute(pipe))

	c.Exec(plumber.UnaryRouter[[]createOperation]{
		SourceTarget: "queue",
		SinkTarget:   "batch",
		Capacity:     1,
	}.PreRoute(pipe))

	c.Exec(plumber.UnaryRouter[[]createOperation]{
		SourceTarget: "batch",
		SinkTarget:   "persist",
		Capacity:     1,
	}.PreRoute(pipe))

	pipe.Flow(ctx)

	return queryFactory, c.Error()
}
