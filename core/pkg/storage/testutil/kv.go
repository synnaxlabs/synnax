// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"io"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/pebblekv"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// OpenKV opens an in-memory key-value store that notifies its observers of each commit
// on their own goroutine, as storage.OpenLayer does. Use it to build a storage.Layer by
// hand. The caller must close the returned store.
func OpenKV() kv.DB {
	sCtx, cancel := signal.Isolated()
	return &asyncKV{
		DB:       memkv.New(pebblekv.WithObserver(observe.NewAsync[kv.TxReader](sCtx))),
		shutdown: signal.NewHardShutdown(sCtx, cancel),
	}
}

type asyncKV struct {
	kv.DB
	shutdown io.Closer
}

// Close stops the observer handlers, then closes the store.
func (k *asyncKV) Close() error { return errors.Join(k.shutdown.Close(), k.DB.Close()) }
