package cesium

import (
	"github.com/arya-analytics/cesium/internal/core"
	"github.com/arya-analytics/x/kfs"
	"github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/kv/pebblekv"
	"github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/pebble"
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
//	 // Set custom shutdown options.
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

	// |||||| FILE SYSTEM ||||||

	fs, err := openFS(ctx, o)
	if err != nil {
		shutdown()
		return nil, err
	}

	// |||||| txn ||||||

	if err := maybeOpenKv(o); err != nil {
		shutdown()
		return nil, err
	}

	// |||||| CREATE ||||||

	create, err := startCreate(ctx, createConfig{
		exp:    o.exp,
		logger: o.logger,
		fs:     fs,
		kv:     o.kv.engine,
	})
	if err != nil {
		shutdown()
		return nil, err
	}

	// |||||| RETRIEVE ||||||

	retrieve, err := startRetrieve(ctx, retrieveConfig{
		exp:    o.exp,
		logger: o.logger,
		fs:     fs,
		kv:     o.kv.engine,
	})
	if err != nil {
		shutdown()
		return nil, err
	}

	// |||||| CHANNEL ||||||

	// a kv persisted counter that tracks the number of channels that a gorpDB has created.
	// this is used to autogenerate unique keys for a channel.
	channelKeyCounter, err := kv.NewPersistedCounter(o.kv.engine, []byte(channelCounterKey))
	if err != nil {
		shutdown()
		return nil, err
	}

	return &db{
		kv:                o.kv.engine,
		externalKV:        o.kv.external,
		shutdown:          shutdown,
		create:            create,
		retrieve:          retrieve,
		channelKeyCounter: channelKeyCounter,
		wg:                ctx,
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
	sync.Start(ctx)
	return fs, err
}

func maybeOpenKv(opts *options) error {
	if opts.kv.engine == nil {
		pebbleDB, err := pebble.Open(
			filepath.Join(opts.dirname, kvDirectory),
			&pebble.Options{FS: opts.kv.fs},
		)
		opts.kv.engine = pebblekv.Wrap(pebbleDB)
		return err
	}
	return nil
}
