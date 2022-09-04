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
	"github.com/arya-analytics/x/lock"
	"github.com/arya-analytics/x/override"
	"github.com/arya-analytics/x/queue"
	"github.com/arya-analytics/x/signal"
	"github.com/arya-analytics/x/validate"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"sync"
	"time"
)

// WriteRequest is a request containing a set of segments (segment) to write to the DB.
type WriteRequest struct {
	// Segments is a set of segments to write to the DB.
	Segments []segment.Segment
}

// WriteResponse contains any errors that occurred during the execution of the Create Query.
type WriteResponse struct {
	// Err is any error that occurred during writer execution.
	Err error
}

type StreamWriter = confluence.Segment[WriteRequest, WriteResponse]

// Writer writes segmented telemetry to the DB. A writer must be closed after use. A Writer
// is not goroutine-safe, but it is safe to use multiple writers for different channels
// concurrently.
//
// Writer is asynchronous, meaning that calls to Write will return before segments are
// persisted to disk. The caller can guarantee that all segments have been persisted
// by calling Close.
type Writer interface {
	// Write writes the provided segments to the DB. If the Writer has encountered an
	// operational error, this method will return false, and the caller is expected
	// to close the Writer. After Write returns false, subsequent calls to Write will
	// return false immediately.
	//
	// Segments must have channel keys in the set provided to DB.NewWriter. Segment data
	// must also be valid i.e. it must have non-zero length and be a multiple of the
	// channel's density. All segments must be provided in time-sorted order on a
	// per-channel basis.
	Write(segments []segment.Segment) bool
	// Close closes the Writer and returns any error accumulated during execution. Close
	// will block until all segments have been persisted to the DB. It is not safe
	// to call Close concurrently with any other Writer methods.
	Close() error
}

type writer struct {
	internal  streamWriter
	requests  confluence.Inlet[WriteRequest]
	responses confluence.Outlet[WriteResponse]
	wg        signal.WaitGroup
	_error    error
}

func wrapStreamWriter(internal *streamWriter) *writer {
	sCtx, _ := signal.Background()
	req := confluence.NewStream[WriteRequest]()
	res := confluence.NewStream[WriteResponse]()
	internal.InFrom(req)
	internal.OutTo(res)
	internal.Flow(
		sCtx,
		confluence.CloseInletsOnExit(),
		confluence.CancelOnExitErr(),
	)
	return &writer{internal: *internal, requests: req, responses: res}
}

// Write implements the Writer interface.
func (w writer) Write(segments []segment.Segment) bool {
	if w.error() != nil {
		return false
	}
	w.requests.Inlet() <- WriteRequest{Segments: segments}
	return true
}

// Close implements the Writer interface.
func (w writer) Close() (err error) {
	w.requests.Close()
	for res := range w.responses.Outlet() {
		err = res.Err
	}
	return err
}

func (w writer) error() error {
	select {
	case res := <-w.responses.Outlet():
		w._error = res.Err
	default:
	}
	return w._error
}

type streamWriter struct {
	confluence.AbstractUnarySource[WriteResponse]
	confluence.UnarySink[WriteRequest]
	lock       lock.Keys[ChannelKey]
	keys       []ChannelKey
	metrics    createMetrics
	wg         *sync.WaitGroup
	parser     *createParser
	operations confluence.Inlet[[]createOperationUnary]
}

func newStreamWriter(
	keys []ChannelKey,
	lock lock.Keys[ChannelKey],
	kve kvx.DB,
	metrics createMetrics,
	logger *zap.Logger,
	operations confluence.Inlet[[]createOperationUnary],
) (*streamWriter, error) {
	keys = lo.Uniq(keys)
	if len(keys) == 0 {
		return nil, errors.New("[cesium] - writer opened without keys")
	}
	channels, err := kv.NewChannelService(kve).Get(keys...)
	if err != nil {
		return nil, err
	}
	if !lock.TryLock(keys...) {
		return nil, errors.New("[cesium] - lock already held")
	}

	channelMap := make(map[ChannelKey]channel.Channel)
	for _, ch := range channels {
		channelMap[ch.Key] = ch
	}

	responses := confluence.AbstractUnarySource[WriteResponse]{}
	wg := &sync.WaitGroup{}

	return &streamWriter{
		lock:       lock,
		keys:       keys,
		metrics:    metrics,
		wg:         wg,
		operations: operations,
		parser: &createParser{
			logger:    logger,
			metrics:   metrics,
			header:    kv.NewHeader(kve),
			channels:  channelMap,
			responses: responses,
			wg:        wg,
		},
	}, nil
}

func (s *streamWriter) Flow(_ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(s.Out)

	requestDur := s.metrics.request.Stopwatch()
	requestDur.Start()

	_ctx.Go(func(ctx context.Context) error {
		defer func() {
			requestDur.Stop()
			s.lock.Unlock(s.keys...)
		}()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req, ok := <-s.In.Outlet():
				if !ok {
					s.wg.Wait()
					return nil
				}
				ops, err := s.parser.parse(ctx, req.Segments)
				if err != nil {
					s.Out.Inlet() <- WriteResponse{Err: err}
					continue
				}
				s.wg.Add(len(ops))
				if err := signal.SendUnderContext(ctx, s.operations.Inlet(), ops); err != nil {
					return err
				}
			}
		}
	}, o.Signal...)
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
		SourceTarget: "allocator",
		SinkTarget:   "queue",
	}.MustRoute(pipe)

	plumber.UnaryRouter[[]createOperationUnary]{
		SourceTarget: "queue",
		SinkTarget:   "batch",
	}.MustRoute(pipe)

	plumber.UnaryRouter[[]createOperationSet]{
		SourceTarget: "batch",
		SinkTarget:   "persist",
	}.MustRoute(pipe)

	pipe.Flow(ctx)

	return operations, nil
}
