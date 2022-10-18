package cesium

import (
	"github.com/cockroachdb/pebble"
	"github.com/synnaxlabs/cesium/internal/allocate"
	"github.com/synnaxlabs/cesium/internal/cache"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/kv"
	"github.com/synnaxlabs/cesium/internal/storage"
	"github.com/synnaxlabs/x/kfs"
	"github.com/synnaxlabs/x/kv/pebblekv"
	"github.com/synnaxlabs/x/lock"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
	"path/filepath"
)

const channelCounterKey = "cs-nc"

// Open opens a new DB whose files are stored in the given directory.
// DB can be opened with a variety of options:
//
//		// Open a DB in memory.
//	 cesium.MemBacked()
//
//	 // Open a DB with the provided logger.
//		cesium.WithLogger(zap.NewNop())
//
//		// Bind an alamos.Experiment to register DB metrics.
//		cesium.WithExperiment(alamos.WithCancel("myExperiment"))
//
//		// Override the default shutdown threshold.
//	 cesium.WithShutdownThreshold(time.Second)
//
//	 // SetMultiple custom shutdown options.
//		cesium.WithShutdownOptions()
//
// See each options documentation for more.
func Open(dirname string, opts ...Option) (DB, error) {
	o := newOptions(dirname, opts...)

	ctx, shutdown := signal.Background(
		signal.WithLogger(o.logger),
		signal.WithContextKey("cesium"),
	)

	sugaredL := o.logger.Sugar()
	sugaredL.Infow("opening cesium time series engine", o.logArgs()...)

	fs, err := openFS(ctx, o)
	if err != nil {
		shutdown()
		return nil, err
	}

	store := storage.Wrap(fs)

	if err := maybeOpenKv(o); err != nil {
		shutdown()
		return nil, err
	}

	kvDB, err := kv.Open(o.kv.engine)
	if err != nil {
		return nil, err
	}

	alloc := allocate.New[ChannelKey, core.FileKey](kvDB.NextFile, allocate.DefaultConfig)

	channelCache := cache.WrapChannelEngine(kvDB, kvDB)

	idx := &indexingEngine{
		channelReader: channelCache,
		memSearchers:  make(map[ChannelKey]index.Searcher),
		memWriters:    make(map[ChannelKey]index.Writer),
		storage:       store,
		kvDB:          kvDB,
	}

	return &db{
		kv:          kvDB,
		channels:    channelCache,
		externalKV:  o.kv.external,
		shutdown:    shutdown,
		channelLock: lock.NewKeys[ChannelKey](),
		logger:      o.logger,
		wg:          ctx,
		storage:     store,
		allocator:   alloc,
		indexes:     idx,
	}, nil
}

func openFS(ctx signal.Context, opts *options) (core.FS, error) {
	dirname := filepath.Join(opts.dirname, cesiumDirectory)
	fs, err := kfs.New[core.FileKey](
		dirname,
		opts.fs.opts...,
	)
	sync := &kfs.Sync[core.FileKey]{
		FS:       fs,
		Interval: opts.fs.sync.interval,
		MaxAge:   opts.fs.sync.maxAge,
	}

	go func() {
		for err := range sync.Start(ctx) {
			opts.logger.Error("failed to sync cesium directory", zap.Error(err))
		}
	}()

	return fs, err
}

func maybeOpenKv(opts *options) error {
	if opts.kv.engine == nil {
		pebbleDB, err := pebble.Open(
			filepath.Join(opts.dirname, kvDirectory),
			&pebble.Options{FS: opts.kv.fs},
		)
		pebbleDB.Flush()
		opts.kv.engine = pebblekv.Wrap(pebbleDB)
		return err
	}
	return nil
}
