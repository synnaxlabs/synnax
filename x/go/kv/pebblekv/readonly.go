// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pebblekv

import (
	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
)

// ErrFormatUnsupported is returned by OpenReadOnly when the store's format is too old
// to open without migrating, which a read-only open never does.
var ErrFormatUnsupported = errors.New(
	"store format requires migration; start a Core against it once to migrate",
)

// OpenReadOnly opens the pebble store at dirname in read-only mode and wraps it as a
// kv.DB with observation disabled. The store's format version is never ratcheted:
// ErrFormatUnsupported is returned instead when the format is too old. Writes to the
// returned DB fail. The open still acquires pebble's directory lock, so it fails when
// another process holds the store, and holds off other processes until Close.
func OpenReadOnly(
	dirname string, ins alamos.Instrumentation,
) (kv.DB, error) {
	desc, err := pebble.Peek(dirname, vfs.Default)
	if err != nil {
		return nil, err
	}
	if desc.FormatMajorVersion < pebble.FormatMinSupported {
		return nil, errors.Wrapf(
			ErrFormatUnsupported, "format version %d", desc.FormatMajorVersion,
		)
	}
	log := NewLogger(ins)
	ev := pebble.MakeLoggingEventListener(log)
	db, err := pebble.Open(dirname, &pebble.Options{
		FS:                 vfs.Default,
		ReadOnly:           true,
		FormatMajorVersion: desc.FormatMajorVersion,
		Logger:             log,
		EventListener:      &ev,
	})
	if err != nil {
		return nil, err
	}
	return Wrap(db, DisableObservation()), nil
}
