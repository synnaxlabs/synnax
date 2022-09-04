package cesium

import (
	"github.com/arya-analytics/cesium/internal/core"
	"github.com/arya-analytics/cesium/internal/persist"
	"github.com/arya-analytics/cesium/internal/segment"
	"github.com/arya-analytics/x/alamos"
	"github.com/arya-analytics/x/config"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/plumber"
	"github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/override"
	"github.com/arya-analytics/x/signal"
	"github.com/arya-analytics/x/telem"
	"github.com/arya-analytics/x/validate"
)

type ResponseVariant uint8

const (
	// AckResponse is a response that indicates that an iteration request was acknowledged.
	AckResponse ResponseVariant = iota + 1
	// DataResponse is a response that indicates that an iteration request returned data.
	DataResponse
)

type IterateRequest struct {
	Command IteratorCommand
	Span    telem.TimeSpan
	Range   telem.TimeRange
	Stamp   telem.TimeStamp
}

//go:generate stringer -type=IteratorCommand
type IteratorCommand uint8

func (i IteratorCommand) hasOps() bool { return i <= IterRange }

const (
	IterNext IteratorCommand = iota + 1
	IterPrev
	IterFirst
	IterLast
	IterNextSpan
	IterPrevSpan
	IterRange
	IterValid
	IterError
	IterSeekFirst
	IterSeekLast
	IterSeekLT
	IterSeekGE
)

// IterateResponse is a response containing segments satisfying a Retrieve Query as well as any errors
// encountered during the retrieval.
type IterateResponse struct {
	Counter  int
	Command  IteratorCommand
	Variant  ResponseVariant
	Ack      bool
	Err      error
	Segments []segment.Segment
}

type readConfig struct {
	// exp is used to track metrics for the Retrieve query. See retrieveMetrics for more.
	exp alamos.Experiment
	// fs is the file system for reading segment data from.
	fs core.FS
	// kv is the key-value store for reading segment metadata from.
	kv kv.DB
	// persist used for setting the parameters for persist.Persist thar reads
	// segment data from disk.
	persist persist.Config
}

func (cfg readConfig) Override(other readConfig) readConfig {
	cfg.exp = override.Nil(cfg.exp, other.exp)
	cfg.fs = override.Nil(cfg.fs, other.fs)
	cfg.kv = override.Nil(cfg.kv, other.kv)
	cfg.persist.NumWorkers = override.Numeric(cfg.persist.NumWorkers, other.persist.NumWorkers)
	cfg.persist = cfg.persist.Override(other.persist)
	return cfg
}

func (cfg readConfig) Validate() error {
	v := validate.New("cesium.read")
	validate.NotNil(v, "fs", cfg.fs)
	validate.NotNil(v, "kv", cfg.kv)
	v.Exec(cfg.persist.Validate)
	return v.Error()
}

var defaultReadConfig = readConfig{persist: persist.DefaultConfig}

func startReadPipeline(ctx signal.Context, _cfg ...readConfig) (confluence.Inlet[[]retrieveOperationUnary], error) {
	cfg, err := config.OverrideAndValidate(defaultReadConfig, _cfg...)
	if err != nil {
		return nil, err
	}

	operations := confluence.NewStream[[]retrieveOperationUnary]()
	pipe := plumber.New()
	batch := newRetrieveBatch()
	batch.InFrom(operations)

	// batch groups operations into batches that optimize sequential IO.
	plumber.SetSegment[[]retrieveOperationUnary, []retrieveOperationSet](
		pipe,
		"batch",
		batch,
	)

	// persist executes batched operations on disk.
	plumber.SetSink[[]retrieveOperationSet](
		pipe,
		"persist",
		persist.New[core.FileKey, retrieveOperationSet](cfg.fs, cfg.persist),
	)

	plumber.UnaryRouter[[]retrieveOperationSet]{
		SourceTarget: "batch",
		SinkTarget:   "persist",
	}.MustRoute(pipe)

	pipe.Flow(ctx)

	return operations, nil
}
