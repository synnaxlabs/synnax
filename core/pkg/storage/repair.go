// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package storage

import (
	"context"
	"path/filepath"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv/pebblekv"
)

// errOldLayout is the refusal of a repair of data on the old storage layout.
var errOldLayout = errors.New(
	"nothing to repair: this Core uses the old storage layout",
)

// Repair repairs the damage of the log of the telemetry in the data directory of cfg,
// as cesium.Repair does: it gives the report to confirm, and writes the repair only
// when confirm returns true. It holds the lock of the directory, and refuses data that
// a Core other than cfg.Version opened last. It refuses a Core with the pointer log
// off, and data on the old storage layout, with an error that says there is nothing to
// repair.
func Repair(
	ctx context.Context,
	cfg LayerConfig,
	confirm func(cesium.Report) bool,
) (rep cesium.Report, err error) {
	if cfg, err = config.New(defaultLayerConfig(), cfg); err != nil {
		return cesium.Report{}, err
	}
	if !*cfg.PointerLogEnabled {
		return cesium.Report{}, errOldLayout
	}
	kvFS, tsFS := openFileSystems(cfg)
	lock, err := acquireLock(cfg.Dirname, kvFS)
	if err != nil {
		return cesium.Report{}, err
	}
	defer func() { err = errors.Join(err, lock.Close()) }()
	if err = CheckOpenedLast(cfg.Instrumentation, cfg.Dirname, cfg.Version,
		"the repair"); err != nil {
		return cesium.Report{}, err
	}
	held, err := readHeldIdentity(cfg, kvFS)
	if err != nil {
		return cesium.Report{}, err
	}
	rep, err = cesium.Repair(
		ctx,
		filepath.Join(cfg.Dirname, cesiumDir),
		confirm,
		append(tsOptions(cfg, tsFS), cesium.WithIdentity(held))...,
	)
	if errors.Is(err, errors.ErrUnsupported) {
		return cesium.Report{}, errOldLayout
	}
	return rep, err
}

// readHeldIdentity returns the identity of the Cesium database that the metadata store
// of cfg on fs holds, which it opens read-only, or 0 when it holds none.
func readHeldIdentity(cfg LayerConfig, fs vfs.FS) (id uint64, err error) {
	logger := pebblekv.NewLogger(cfg.Child("kv"))
	db, err := pebble.Open(filepath.Join(cfg.Dirname, kvDir), &pebble.Options{
		FS:       fs,
		ReadOnly: true,
		Logger:   logger,
	})
	if err != nil {
		return 0, errors.Wrapf(err, "failed to open the key-value store of %s",
			cfg.Dirname)
	}
	meta := pebblekv.Wrap(db, logger)
	defer func() { err = errors.Join(err, meta.Close()) }()
	return readTSIdentity(meta)
}

// CheckOpenedLast refuses the data in dirname when a Core other than version vsn
// opened it last, so that a command on the data starts no update. After a failed
// update to vsn, the refusal names the version that the data is back at. A dev build
// checks nothing. command names the command, such as "the purge", in the refusal.
func CheckOpenedLast(ins alamos.Instrumentation, dirname, vsn, command string) error {
	if version.IsDev(vsn) {
		return nil
	}
	backups, err := NewBackups(BackupsConfig{Instrumentation: ins, Dirname: dirname})
	if err != nil {
		return err
	}
	state, err := backups.UpdateState(vsn)
	if err != nil {
		return err
	}
	switch state {
	case UpdateCommitted:
		return nil
	case UpdateUndone:
		back, err := backups.readVersion()
		if err != nil {
			return err
		}
		at := "Core " + back
		if back == UnknownVersion {
			at = "the Core before it"
		}
		return errors.Newf(
			"The update to Core %s failed, and the data is back at %s. Run %s of %s on "+
				"the data, then start Core %s.",
			vsn,
			at,
			command,
			at,
			vsn,
		)
	case UpdateNotStarted, UpdatePending:
		return errors.Newf(
			"Core %s did not open the data last. Start Core %s on the data, stop it, "+
				"and run %s again.",
			vsn,
			vsn,
			command,
		)
	}
	return errors.Newf("unknown update state %d", state)
}
