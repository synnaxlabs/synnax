// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package storage provides entities for managing node local storage. Synnax implements
// two database classes for storing its data:
//
//  1. A key-value store (implementing the kv.DB interface) for storing cluster wide
//     metadata.
//  2. A time-series engine (implementing the TS interface) for writing frames of
//     telemetry.
//
// It's important to note that the storage package does NOT manage any sort of
// distributed storage implementation.
package storage

import (
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errutil"
	"github.com/synnaxlabs/x/gorp"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/pebblekv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type (
	// KVEngine is an enumeration of  the available key-value storage engines synnax can use.
	KVEngine uint8
	// TSEngine is an enumeration of the available time-series storage engines delta can use.
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
// used to read and write data. A Store must be closed when it is no longer in use.
type Store struct {
	// Config is the configuration for the storage provided to Open.
	Config
	// KV is the key-value store for the node.
	KV kv.DB
	// TS is the time-series engine for the node.
	TS TS
	// lock is the lock held on the storage directory.
	lock io.Closer
}

// Gorpify returns a gorp.DB that can be used to interact with the storage key-value store.
func (s *Store) Gorpify() *gorp.DB {
	return gorp.Wrap(
		s.KV,
		gorp.WithEncoderDecoder(&binary.TracingEncoderDecoder{
			Level:           alamos.Debug,
			Instrumentation: s.Instrumentation,
			EncoderDecoder:  &binary.MsgPackEncoderDecoder{},
		}),
	)
}

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
	c.Exec(s.lock.Close)
	return c.Error()
}

// Config is used to configure delta's storage layer.
type Config struct {
	alamos.Instrumentation
	// Dirname defines the root directory the Store resides. The given directory
	// shouldn't be used by another process while the node is running.
	Dirname string
	// Perm is the file permissions to use for the storage directory.
	Perm fs.FileMode
	// MemBacked defines whether the node should use a memory-backed file system.
	MemBacked *bool
	// KVEngine is the key-value engine storage will use.
	KVEngine KVEngine
	// TSEngine is the time-series engine storage will use.
	TSEngine TSEngine
}

var _ config.Config[Config] = Config{}

// Override implements Config.
func (cfg Config) Override(other Config) Config {
	cfg.Dirname = override.String(cfg.Dirname, other.Dirname)
	cfg.Perm = override.Numeric(cfg.Perm, other.Perm)
	cfg.KVEngine = override.Numeric(cfg.KVEngine, other.KVEngine)
	cfg.TSEngine = override.Numeric(cfg.TSEngine, other.TSEngine)
	cfg.MemBacked = override.Nil(cfg.MemBacked, other.MemBacked)
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	if *cfg.MemBacked {
		cfg.Dirname = ""
	}
	return cfg
}

// Validate implements Config.
func (cfg Config) Validate() error {
	v := validate.New("storage")
	v.Ternaryf(!*cfg.MemBacked && cfg.Dirname == "", "dirname must be set")
	v.Ternaryf(!lo.Contains(kvEngines, cfg.KVEngine), "invalid key-value engine %s", cfg.KVEngine)
	v.Ternaryf(!lo.Contains(tsEngines, cfg.TSEngine), "invalid time-series engine %s", cfg.TSEngine)
	v.Ternary(cfg.Perm == 0, "insufficient permission bits on directory")
	return v.Error()
}

// Report implements the alamos.ReportProvider interface.
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
	Perm:      xfs.OS_USER_RWX,
	MemBacked: config.Bool(false),
	KVEngine:  PebbleKV,
	TSEngine:  CesiumTS,
}

// Open opens a new Store with the given Config. Open acquires a lock on the directory
// specified in the Config. If the lock cannot be acquired, Open returns an error.
// The lock is released when the Store is/closed. Store MUST be closed when it is no
// longer in use.
func Open(cfg Config) (s *Store, err error) {
	cfg, err = config.New(DefaultConfig, cfg)
	if err != nil {
		return nil, err
	}

	s = &Store{Config: cfg}

	s.L.Info("opening storage", cfg.Report().ZapFields()...)

	// Open our two file system implementations. We use VFS for acquiring the directory
	// lock and for the key-value store. We use XFS for the time-series engine, as we
	// need seekable file handles.
	baseVFS, baseXFS := openBaseFS(cfg)

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
	s.lock = releaser

	// Open the key-value storage engine.
	if s.KV, err = openKV(cfg, baseVFS); err != nil {
		return s, errors.CombineErrors(err, s.lock.Close())
	}

	// Open the time-series engine.
	if s.TS, err = openTS(cfg, baseXFS); err != nil {
		err = errors.CombineErrors(err, s.KV.Close())
		return s, errors.CombineErrors(err, s.lock.Close())
	}

	return s, nil
}

const (
	kvDirname     = "kv"
	lockFileName  = "LOCK"
	cesiumDirname = "cesium"
)

func openBaseFS(cfg Config) (vfs.FS, xfs.FS) {
	if *cfg.MemBacked {
		return vfs.NewMem(), xfs.NewMem()
	} else {
		return vfs.Default, xfs.Default
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

Synnax requires the storage directory to have at least

%v

permissions.
`

func validateSufficientDirPermissions(cfg Config) error {
	stat, err := os.Stat(cfg.Dirname)
	if err != nil {
		return err
	}
	// We need the directory to have at least the permissions set in ServiceConfig.Perm.
	if !xfs.CheckSufficientPermissions(stat.Mode().Perm(), cfg.Perm) {
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

Is there another Synnax node using the same directory?
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
	if cfg.KVEngine != PebbleKV {
		{
			return nil, errors.Newf("[storage]- unsupported key-value engine: %s", cfg.TSEngine)
		}
	}
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

func openTS(cfg Config, fs xfs.FS) (TS, error) {
	if cfg.TSEngine != CesiumTS {
		return nil, errors.Newf("[storage] - unsupported time-series engine: %s", cfg.TSEngine)
	}
	dirname := filepath.Join(cfg.Dirname, cesiumDirname)
	return cesium.Open(
		dirname,
		cesium.WithFS(fs),
		cesium.WithInstrumentation(cfg.Instrumentation.Child("cesium")),
	)
}
