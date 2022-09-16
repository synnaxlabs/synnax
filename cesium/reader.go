package cesium

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/persist"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
)

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

func startReadPipeline(ctx signal.Context, _cfg ...readConfig) (confluence.Inlet[[]readOperation], error) {
	cfg, err := config.OverrideAndValidate(defaultReadConfig, _cfg...)
	if err != nil {
		return nil, err
	}

	operations := confluence.NewStream[[]readOperation]()
	pipe := plumber.New()
	batch := newRetrieveBatch()
	batch.InFrom(operations)

	// batch groups operations into batches that optimize sequential IO.
	plumber.SetSegment[[]readOperation, []retrieveOperationSet](
		pipe,
		"batch",
		batch,
	)

	pst, err := persist.New[core.FileKey, retrieveOperationSet](cfg.fs, cfg.persist)
	if err != nil {
		return nil, err
	}

	// persist executes batched operations on disk.
	plumber.SetSink[[]retrieveOperationSet](pipe, "persist", pst)

	plumber.UnaryRouter[[]retrieveOperationSet]{
		SourceTarget: "batch",
		SinkTarget:   "persist",
		Capacity:     1,
	}.MustRoute(pipe)

	pipe.Flow(ctx)

	return operations, nil
}
