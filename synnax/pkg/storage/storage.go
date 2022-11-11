// Package storage provides entities for managing node local stores. Delta uses two
// database classes for storing its data:
//
//  1. Key key-value store (implementing the kv.DB interface) for storing cluster wide
//     metadata.
//  2. Key time-series engine (implementing the cesium.DB interface) for writing chunks
//     of time-series data.
//
// It's important to node the storage package does NOT manage any sort of distributed
// storage implementation.
package storage

import (
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errutil"
	"github.com/synnaxlabs/x/fsutil"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kfs"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/pebblekv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type (
	// KVEngine represents the available key-value storage engines delta can use.
	KVEngine uint8
	// TSEngine represents the available time-series storage engines delta can use.
	TSEngine uint8
)

//go:generate stringer -type=KVEngine
const (
	// PebbleKV uses cockroach's pebble key-value store.
	PebbleKV KVEngine = iota + 1
)

var kvEngines = []KVEngine{PebbleKV}

//go:generate stringer -type=TSEngine
const (
	// CesiumTS uses Synnax's cesium time-series engine.
	CesiumTS TSEngine = iota + 1
)

var tsEngines = []TSEngine{CesiumTS}

// Store represents a node's local storage. The provided KV and TS engines can be
// used to read and write data. Key Store must be closed when it is no longer in use.
type Store struct {
	// Config is the configuration for the storage provided to Open.
	Config Config
	// KV is the key-value store for the node.
	KV kv.DB
	// TS is the time-series engine for the node.
	TS TS
	// releaseLock is a function that releases the lock on the storage file system.
	releaseLock func() error
}

// Gorpify returns a gorp.DB that can be used to interact with the storage key-value store.
func (s *Store) Gorpify() *gorp.DB { return gorp.Wrap(s.KV) }

// Close closes the Store, releasing the lock on the storage directory. Close
// MUST be called when the Store is no longer in use. The caller must ensure that
// all processes interacting the Store have finished before calling Close.
func (s *Store) Close() error {
	// We execute with aggregation here to ensure that we close all engines and release
	// the lock regardless if one engine fails to close. This may cause unexpected
	// behavior in the future, so we need to track it.
	c := errutil.NewCatch(errutil.WithAggregation())
	c.Exec(s.TS.Close)
	c.Exec(s.KV.Close)
	c.Exec(s.releaseLock)
	return c.Error()
}

// Config is used to configure delta's storage layer.
type Config struct {
	// Dirname defines the root directory the Store resides. The given directory
	// shouldn't be used by another process while the node is running.
	Dirname string
	// Perm is the file permissions to use for the storage directory.
	Perm fs.FileMode
	// MemBacked defines whether the node should use a memory-backed file system.
	MemBacked *bool
	// Logger is the logger used by the node.
	Logger *zap.Logger
	// Experiment is the experiment used by the node for metrics, reports, and tracing.
	Experiment alamos.Experiment
	// KVEngine is the key-value engine storage will use.
	KVEngine KVEngine
	// TSEngine is the time-series engine storage will use.
	TSEngine TSEngine
}

var _ config.Config[Config] = Config{}

func (cfg Config) Override(other Config) Config {
	cfg.Dirname = override.String(cfg.Dirname, other.Dirname)
	cfg.Perm = override.Numeric(cfg.Perm, other.Perm)
	cfg.Experiment = override.Nil(cfg.Experiment, other.Experiment)
	cfg.Logger = override.Nil(cfg.Logger, other.Logger)
	cfg.KVEngine = override.Numeric(cfg.KVEngine, other.KVEngine)
	cfg.TSEngine = override.Numeric(cfg.TSEngine, other.TSEngine)
	cfg.MemBacked = override.Nil(cfg.MemBacked, other.MemBacked)
	return cfg
}

func (cfg Config) Validate() error {
	if !*cfg.MemBacked && cfg.Dirname == "" {
		return errors.Wrap(validate.Error, "[storage] - dirname must be set")
	}

	if !lo.Contains[KVEngine](kvEngines, cfg.KVEngine) {
		return errors.Wrap(validate.Error, "[storage] - invalid key-value engine")
	}

	if !lo.Contains[TSEngine](tsEngines, cfg.TSEngine) {
		return errors.Wrap(validate.Error, "[storage] - invalid time-series engine")
	}

	if cfg.Perm == 0 {
		return errors.Wrap(validate.Error, "[storage] - insufficient permission bits configured")
	}

	return nil
}

func (cfg Config) Report() alamos.Report {
	return alamos.Report{
		"dirname":     cfg.Dirname,
		"permissions": cfg.Perm,
		"memBacked":   cfg.MemBacked,
		"kvEngine":    cfg.KVEngine.String(),
		"tsEngine":    cfg.TSEngine.String(),
	}
}

// DefaultConfig returns the default configuration for the storage layer.
var DefaultConfig = Config{
	Perm:       fsutil.OS_USER_RWX,
	MemBacked:  config.BoolPointer(false),
	Logger:     zap.NewNop(),
	Experiment: nil,
	KVEngine:   PebbleKV,
	TSEngine:   CesiumTS,
}

// Open opens a new Store with the given Config. Open acquires a lock on the directory
// specified in the Config. If the lock cannot be acquired, Open returns an error.
// The lock is released when the Store is/closed. Store MUST be closed when it is no
// longer in use.
func Open(cfg Config) (s *Store, err error) {
	cfg, err = config.OverrideAndValidate(DefaultConfig, cfg)
	if err != nil {
		return nil, err
	}

	s = &Store{}

	log := cfg.Logger.Sugar()
	log.Infow("opening storage", cfg.Report().LogArgs()...)

	// Open our two file system implementations. We use VFS for acquiring the directory
	// lock and for the key-value store. We use KFS for the time-series engine, as we
	// need seekable file handles. This is
	baseVFS, baseKFS := openBaseFS(cfg)

	// Configure our storage directory with the correct permissions.
	if err := configureStorageDir(cfg, baseVFS); err != nil {
		return s, err
	}

	// TryLock the lock on the storage directory. If any other delta node is using the
	// same directory we return an error to the client.
	releaser, err := acquireLock(cfg, baseVFS)
	if err != nil {
		return s, err
	}
	// Allow the caller to release the lock when they finish using the storage.
	s.releaseLock = releaser.Close

	// Open the key-value storage engine.
	if s.KV, err = openKV(cfg, baseVFS); err != nil {
		return s, errors.CombineErrors(err, s.releaseLock())
	}

	// Open the time-series engine.
	if s.TS, err = openTS(cfg, baseKFS, baseVFS); err != nil {
		return s, errors.CombineErrors(err, s.releaseLock())
	}

	return s, nil
}

const (
	kvDirname     = "kv"
	lockFileName  = "LOCK"
	cesiumDirname = "cesium"
)

func openBaseFS(cfg Config) (vfs.FS, kfs.BaseFS) {
	if *cfg.MemBacked {
		return vfs.NewMem(), kfs.NewMem()
	} else {
		return vfs.Default, kfs.NewOS()
	}
}

func configureStorageDir(cfg Config, vfs vfs.FS) error {
	if err := vfs.MkdirAll(cfg.Dirname, cfg.Perm); err != nil {
		return errors.Wrapf(err, "failed to create storage directory %s", cfg.Dirname)
	}
	if !*cfg.MemBacked {
		return validateSufficientDirPermissions(cfg)
	}
	return nil
}

const insufficientDirPermissions = `
Existing storage directory

%s

has permissions

%v

Delta requires the storage directory to have at least

%v

permissions.
`

func validateSufficientDirPermissions(cfg Config) error {
	stat, err := os.Stat(cfg.Dirname)
	if err != nil {
		return err
	}
	// We need the directory to have at least the permissions set in Config.Perm.
	if !fsutil.CheckSufficientPermissions(stat.Mode(), cfg.Perm) {
		return errors.Newf(
			insufficientDirPermissions,
			cfg.Dirname,
			stat.Mode().Perm(),
			cfg.Perm,
		)
	}
	return nil
}

const failedToAcquireLockMsg = `
Failed to acquire lock on storage directory

	%s

Is there another Delta node using the same directory?
`

func acquireLock(cfg Config, fs vfs.FS) (io.Closer, error) {
	fName := filepath.Join(cfg.Dirname, lockFileName)
	release, err := fs.Lock(fName)
	if err == nil {
		return release, nil
	}
	return release, errors.Wrapf(err, failedToAcquireLockMsg, cfg.Dirname)
}

func openKV(cfg Config, fs vfs.FS) (kv.DB, error) {
	dirname := filepath.Join(cfg.Dirname, kvDirname)
	db, err := pebble.Open(dirname, &pebble.Options{FS: fs})
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"[storage] - failed to open key-value store in %s",
			dirname,
		)
	}
	return pebblekv.Wrap(db), err
}

func openTS(cfg Config, fs kfs.BaseFS, vfs vfs.FS) (TS, error) {
	if cfg.TSEngine != CesiumTS {
		return nil, errors.Newf("[storage] - unsupported time-series engine: %v", cfg.TSEngine)
	}
	dirname := filepath.Join(cfg.Dirname, cesiumDirname)
	return cesium.Open(
		dirname,
		cesium.WithFS(vfs, fs),
		cesium.WithLogger(cfg.Logger),
		cesium.WithExperiment(cfg.Experiment),
	)
}
