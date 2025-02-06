// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package memkv implements an in-memory key value store using cockroachdb's pebble storage engine.
// It's particularly useful for testing scenarios.
package memkv

import (
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/pebblekv"
)

// New opens a new in-memory key-value store implementing the kv.DB interface.
func New() kv.DB {
	db, err := pebble.Open("", &pebble.Options{FS: vfs.NewMem()})
	if err != nil {
		panic(err)
	}
	return pebblekv.Wrap(db)
}
