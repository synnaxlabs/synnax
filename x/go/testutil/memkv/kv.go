// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// All included pebble code is copyrighted by the cockroachdb team, and is licensed
// under the BSD 3-Clause License. See the repository file licenses/BSD-3-Clause.txt for
// more information.

// Package memkv implements an in-memory key value store using cockroachdb's pebble
// storage engine. It's particularly useful for testing scenarios.
package memkv

import (
	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/pebblekv"
)

// New opens a new in-memory key-value store implementing the kv.DB interface.
func New() kv.DB {
	return pebblekv.Wrap(lo.Must(pebble.Open("", &pebble.Options{
		FS:     vfs.NewMem(),
		Logger: pebblekv.NewNoopLogger(),
	})))
}
