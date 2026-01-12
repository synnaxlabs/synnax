// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"context"
	"sync"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/binary"
	"go.uber.org/zap"
)

// Subscriber is used to flush an observable that flushes changes
// to an underlying key-value store.
type Subscriber[S any] struct {
	alamos.Instrumentation
	// Key is the key to flush the contents of the observable into.
	Key []byte
	// Store is the store to flush the contents of the observable into.
	Store Writer
	// MinInterval specifies the minimum interval between flushes. If the observable
	// updates more quickly than min interval, the Subscriber will not flush the
	// contents.
	MinInterval time.Duration
	// LastFlush stores the last time the observable was flushed.
	LastFlush time.Time
	// Encoder is the encoder to use when flushing the contents of the state
	Encoder binary.Encoder
	// mu is used to prevent multiple flushes from racing over each other.
	mu sync.Mutex
}

// Flush is the handler to bind to the Observable.
func (f *Subscriber[S]) Flush(ctx context.Context, state S) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if time.Since(f.LastFlush) < f.MinInterval {
		return
	}
	f.LastFlush = time.Now()
	go f.FlushSync(ctx, state)
}

// FlushSync synchronously flushes the given state to the store.
func (f *Subscriber[S]) FlushSync(ctx context.Context, state S) {
	if err := f.Store.Set(ctx, f.Key, lo.Must(f.Encoder.Encode(ctx, state))); err != nil {
		f.L.Error("failed to flush", zap.Error(err))
	}
}
