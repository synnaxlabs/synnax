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
	"github.com/arya-analytics/x/config"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/plumber"
	kvx "github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/override"
	"github.com/arya-analytics/x/query"
	"github.com/arya-analytics/x/queue"
	"github.com/arya-analytics/x/signal"
	"github.com/arya-analytics/x/validate"
	"github.com/cockroachdb/errors"
	"sync"
	"time"
)

// WriteRequest is a request containing a set of segments (segment) to write to the DB.
type WriteRequest struct {
	Segments []segment.Segment
}

// WriteResponse contains any errors that occurred during the execution of the Create Query.
type WriteResponse struct {
	Error error
}

func (c Create) Stream(ctx context.Context) (chan<- WriteRequest, <-chan WriteResponse, error) {
	keys := channel.GetKeys(c)
	requests := confluence.NewStream[WriteRequest](0)
	responses := confluence.NewStream[WriteResponse](0)

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

	res := confluence.AbstractUnarySource[WriteResponse]{}
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
					responses.Inlet() <- WriteResponse{Error: err}
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

type writeConfig struct {
	// exp is used to track metrics for the Create query. See createMetrics for all the recorded values.
	exp alamos.Experiment
	// fs is the file system for writing segment data to.
	fs core.FS
	// kv is the key-value store for writing segment metadata to.
	kv kvx.DB
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

var _ config.Config[writeConfig] = writeConfig{}

func (cfg writeConfig) Override(other writeConfig) writeConfig {
	cfg.exp = override.Nil(cfg.exp, other.exp)
	cfg.fs = override.Nil(cfg.fs, other.fs)
	cfg.kv = override.Nil(cfg.kv, other.kv)
	cfg.allocate = cfg.allocate.Override(other.allocate)
	cfg.persist = cfg.persist.Override(other.persist)
	cfg.debounce = cfg.debounce.Override(other.debounce)
	return cfg
}

func (cfg writeConfig) Validate() error {
	v := validate.New("cesium.writeConfig")
	validate.NotNil(v, "fs", cfg.fs)
	validate.NotNil(v, "kv", cfg.kv)
	v.Exec(cfg.allocate.Validate)
	v.Exec(cfg.persist.Validate)
	v.Exec(cfg.debounce.Validate)
	return v.Error()
}

var defaultWriteConfig = writeConfig{
	allocate: allocate.DefaultConfig,
	persist:  persist.DefaultConfig,
	debounce: queue.DebounceConfig{
		FlushInterval:  10 * time.Millisecond,
		FlushThreshold: 100,
	},
}

func startCreate(ctx signal.Context, _cfg ...writeConfig) (confluence.Inlet[[]createOperationUnary], error) {
	cfg, err := config.OverrideAndValidate(defaultWriteConfig, _cfg...)
	if err != nil {
		return nil, err
	}

	// a kv persisted counter that tracks the number of files that a DB has created.
	// The segment allocator uses it to determine the next file to open.
	fCount, err := openFileCounter(cfg.kv)
	if err != nil {
		return nil, err
	}

	operations := confluence.NewStream[[]createOperationUnary]()
	pipe := plumber.New()
	allocator := newAllocator(fCount, cfg.allocate)
	allocator.InFrom(operations)

	// allocator allocates segments to files.
	plumber.SetSegment[[]createOperationUnary, []createOperationUnary](
		pipe,
		"allocator",
		allocator,
	)

	// queue 'debounces' operations so that they can be flushed to disk in efficient
	// batches.
	plumber.SetSegment[[]createOperationUnary, []createOperationUnary](
		pipe,
		"queue",
		&queue.Debounce[createOperationUnary]{Config: cfg.debounce},
	)

	// batch groups operations into batches that are more efficient upon retrieval.
	plumber.SetSegment[[]createOperationUnary, []createOperationSet](pipe, "batch", newCreateBatch())

	// persist executes batched operations to disk.
	plumber.SetSink[[]createOperationSet](
		pipe,
		"persist",
		persist.New[core.FileKey, createOperationSet](cfg.fs, cfg.persist),
	)

	plumber.UnaryRouter[[]createOperationUnary]{
		SourceTarget: "query",
		SinkTarget:   "allocator",
		Capacity:     1,
	}.MustRoute(pipe)

	plumber.UnaryRouter[[]createOperationUnary]{
		SourceTarget: "allocator",
		SinkTarget:   "queue",
		Capacity:     1,
	}.MustRoute(pipe)

	plumber.UnaryRouter[[]createOperationUnary]{
		SourceTarget: "queue",
		SinkTarget:   "batch",
		Capacity:     1,
	}.MustRoute(pipe)

	plumber.UnaryRouter[[]createOperationSet]{
		SourceTarget: "batch",
		SinkTarget:   "persist",
		Capacity:     1,
	}.MustRoute(pipe)

	pipe.Flow(ctx)

	return operations, nil
}
