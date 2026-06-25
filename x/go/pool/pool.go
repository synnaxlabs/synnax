// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pool

import (
	"io"
	"sync"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/types"
)

// ErrClosed is returned by Acquire after Close has been called.
//
// Acquiring from a closed pool is a programming error: the contract is that the caller
// stops using the pool before calling Close. ErrClosed exists to surface that bug
// deterministically rather than silently allocate an untracked adapter.
var ErrClosed = errors.New("pool: closed")

// Adapter is a poolable resource (such as a network connection) that a Pool caches and
// hands out to callers. An Adapter tracks how many callers currently hold it so it is
// only torn down once nothing is using it.
type Adapter interface {
	// Closer releases the underlying resource. It is called by Pool.Close.
	io.Closer
	// Healthy reports whether the adapter is still usable. The Pool discards adapters
	// that report false instead of handing them out.
	Healthy() bool
	// Acquire records that an additional caller now holds the adapter.
	Acquire() error
	// Release records that one caller is done with the adapter.
	Release()
}

// Factory opens new Adapters on demand. A Pool calls Open when it has no healthy cached
// adapter for a requested key.
type Factory[K comparable, A Adapter] interface {
	// Open opens a new adapter for the given key.
	Open(K) (A, error)
}

// Pool caches and reuses Adapters keyed by K, opening new ones via a Factory only when
// no healthy adapter is already cached for a key. Implementations are safe for
// concurrent use.
type Pool[K comparable, A Adapter] interface {
	// Acquire returns an adapter for the given key, creating one via the Factory if no
	// healthy adapter is already cached.
	//
	// Calling Acquire after Close is a programming error: the contract is that the
	// caller stops using the pool before Close runs. As a safety net for buggy callers,
	// Acquire returns ErrClosed instead of silently allocating an adapter that nothing
	// will ever close.
	Acquire(K) (A, error)
	// Closer closes every adapter the pool has opened. The pool must not be used after
	// Close returns.
	io.Closer
}

// Open returns a Pool that opens adapters on demand via factory.
func Open[K comparable, A Adapter](factory Factory[K, A]) Pool[K, A] {
	return &pool[K, A]{pool: make(map[K][]A), factory: factory}
}

type pool[K comparable, A Adapter] struct {
	factory Factory[K, A]
	pool    map[K][]A
	mu      sync.Mutex
}

func (p *pool[K, A]) Acquire(key K) (A, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pool == nil {
		return types.Zero[A](), ErrClosed
	}
	if adapters, ok := p.pool[key]; ok {
		for _, adapter := range adapters {
			if adapter.Healthy() {
				if err := adapter.Acquire(); err != nil {
					return types.Zero[A](), err
				}
				return adapter, nil
			}
		}
	}
	return p.open(key)
}

func (p *pool[K, A]) Close() error {
	p.mu.Lock()
	adapters := p.pool
	p.pool = nil
	p.mu.Unlock()

	// Adapter teardowns happen outside the lock so a slow Close (e.g. gRPC client
	// connection shutdown waiting on its internal WaitGroup) does not block concurrent
	// callers and cannot deadlock if an adapter's Close path ever calls back into the
	// pool.
	var err error
	for _, group := range adapters {
		for _, adapter := range group {
			err = errors.Combine(err, adapter.Close())
		}
	}
	return err
}

func (p *pool[K, A]) open(key K) (A, error) {
	a, err := p.factory.Open(key)
	if err != nil {
		return types.Zero[A](), err
	}
	p.pool[key] = append(p.pool[key], a)
	return a, nil
}
