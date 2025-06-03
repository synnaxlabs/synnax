// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// All included pebble code is copyrighted by the cockroachdb team, and is licensed under
// the BSD 3-Clause License. See the repository file license/BSD-3-Clause.txt for more
// information.

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
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/layer"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/pebblekv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
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
	// CesiumTS uses synnax's cesium time-series engine.
	CesiumTS TSEngine = iota + 1
)

var tsEngines = []TSEngine{CesiumTS}

// Layer represents a node's local storage. The provided KV and TS engines can be
// used to read and write data. A Layer must be closed when it is no longer in use.
type Layer struct {
	// Config is the configuration for the storage provided to Open.
	Config
	// KV is the key-value store for the node.
	KV kv.DB
	// TS is the time-series engine for the node.
	TS     *cesium.DB
	closer xio.MultiCloser
}

// Close closes the Layer, releasing the lock on the storage directory. Close
// MUST be called when the Layer is no longer in use. The caller must ensure that
// all processes interacting the Layer have finished before calling Close.
func (s *Layer) Close() error { return s.closer.Close() }

// Config is used to configure delta's storage layer.
type Config struct {
	alamos.Instrumentation
	// Dirname defines the root directory the Layer resides. The given directory
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

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig returns the default configuration for the storage layer.
	DefaultConfig = Config{
		Perm:      xfs.OS_USER_RWX,
		MemBacked: config.False(),
		KVEngine:  PebbleKV,
		TSEngine:  CesiumTS,
	}
)

// Override implements Config.
func (cfg Config) Override(other Config) Config {
	cfg.Dirname = override.String(cfg.Dirname, other.Dirname)
	cfg.Perm = override.Numeric(cfg.Perm, other.Perm)
	cfg.KVEngine = override.Numeric(cfg.KVEngine, other.KVEngine)
	cfg.TSEngine = override.Numeric(cfg.TSEngine, other.TSEngine)
	cfg.MemBacked = override.Nil(cfg.MemBacked, other.MemBacked)
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	if cfg.MemBacked != nil && *cfg.MemBacked {
		cfg.Dirname = ""
	}
	return cfg
}

// Validate implements Config.
func (cfg Config) Validate() error {
	v := validate.New("storage")
	v.Ternaryf("dirname", !*cfg.MemBacked && cfg.Dirname == "", "dirname must be set")
	v.Ternaryf("kvEngine", !lo.Contains(kvEngines, cfg.KVEngine), "invalid key-value engine %s", cfg.KVEngine)
	v.Ternaryf("tsEngine", !lo.Contains(tsEngines, cfg.TSEngine), "invalid time-series engine %s", cfg.TSEngine)
	v.Ternary("permissions", cfg.Perm == 0, "insufficient permission bits on directory")
	return v.Error()
}

// Report implements the alamos.ReportProvider interface.
func (cfg Config) Report() alamos.Report {
	return alamos.Report{
		"dirname":     cfg.Dirname,
		"permissions": cfg.Perm,
		"mem_backed":  cfg.MemBacked,
		"kv_engine":   cfg.KVEngine.String(),
		"ts_engine":   cfg.TSEngine.String(),
	}
}

// Open opens a new Layer with the given Config. Open acquires a lock on the directory
// specified in the Config. If the lock cannot be acquired, Open returns an error.
// The lock is released when the Layer is/closed. Layer MUST be closed when it is no
// longer in use.
func Open(ctx context.Context, cfg Config) (*Layer, error) {
	cfg, err := config.New(DefaultConfig, cfg)
	if err != nil {
		return nil, err
	}
	l := &Layer{Config: cfg}
	cleanup, ok := layer.NewOpener(ctx, &err, &l.closer)
	defer cleanup()

	if *cfg.MemBacked {
		l.L.Info("starting with memory-backed storage. no data will be persisted")
	} else {
		l.L.Info("starting in directory", zap.String("dirname", cfg.Dirname))
	}
	l.L.Debug("config", cfg.Report().ZapFields()...)

	// Open our two file system implementations. We use VFS for acquiring the directory
	// lock and for the key-value store. We use XFS for the time-series engine, as we
	// need seekable file handles.
	baseVFS, baseXFS := openBaseFS(cfg)

	// Configure our storage directory with the correct permissions.
	if err = configureStorageDir(cfg, baseVFS); !ok(nil) {
		return nil, err
	}

	// Try to lock the storage directory. If any other synnax node is using the
	// same directory, we return an error to the client. We'll also add it to the
	// list of closers to release the lock when the storage layer shuts down.
	var lock io.Closer
	if lock, err = acquireLock(cfg, baseVFS); !ok(lock) {
		return nil, err
	}

	// Open the key-value storage engine.
	if l.KV, err = openKV(cfg, baseVFS); !ok(l.KV) {
		return nil, err
	}

	// Open the time-series engine.
	if l.TS, err = openTS(ctx, cfg, baseXFS); !ok(l.TS) {
		return nil, err
	}
	return l, nil
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
		return nil, errors.Newf("[storage]- unsupported key-value engine: %s", cfg.KVEngine)
	}

	dirname := filepath.Join(cfg.Dirname, kvDirname)
	requiresMigration, err := pebblekv.RequiresMigration(dirname, fs)
	if err != nil {
		return nil, err
	}
	if requiresMigration {
		cfg.Instrumentation.L.Info("existing key-value store requires migration. this may take a moment. Be patient and do not kill this process or risk corrupting data")
		if err := pebblekv.Migrate(dirname); err != nil {
			return nil, err
		}
	}

	db, err := pebble.Open(dirname, &pebble.Options{
		FS:                 fs,
		FormatMajorVersion: pebble.FormatNewest,
	})
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"[storage] - failed to open key-value store in %s",
			dirname,
		)
	}
	return pebblekv.Wrap(db), err
}

func openTS(ctx context.Context, cfg Config, fs xfs.FS) (*ts.DB, error) {
	if cfg.TSEngine != CesiumTS {
		return nil, errors.Newf("[storage] - unsupported time-series engine: %s", cfg.TSEngine)
	}
	return ts.Open(ctx, ts.Config{
		Instrumentation: cfg.Instrumentation.Child("ts"),
		Dirname:         filepath.Join(cfg.Dirname, cesiumDirname),
		FS:              fs,
	})
}
