// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package observe

import (
	"context"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/kv"
	"go.uber.org/zap"
	"sync"
	"time"
)

// FlushSubscriber is used to flush an observable that flushes changes
// to an underlying key-value store.
type FlushSubscriber[S any] struct {
	alamos.Instrumentation
	// Key is the key to flush the contents of the observable into.
	Key []byte
	// Store is the store to flush the contents of the observable into.
	Store kv.Writer
	// MinInterval specifies the minimum interval between flushes. If the observable
	// updates more quickly than min interval, the FlushSubscriber will not flush the
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
func (f *FlushSubscriber[S]) Flush(ctx context.Context, state S) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if time.Since(f.LastFlush) < f.MinInterval {
		return
	}
	f.LastFlush = time.Now()
	go f.FlushSync(ctx, state)
}

// FlushSync synchronously flushes the givens tate to the store.
func (f *FlushSubscriber[S]) FlushSync(ctx context.Context, state S) {
	if err := f.Store.Set(ctx, f.Key, lo.Must(f.Encoder.Encode(nil, state))); err != nil {
		f.L.Error("failed to flush", zap.Error(err))
	}
}
