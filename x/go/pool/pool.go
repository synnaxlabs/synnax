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
	"sync"

	"github.com/synnaxlabs/x/errors"
)

// ErrClosed is returned by Acquire after Close has been called.
//
// Acquiring from a closed pool is a programming error: the contract is that
// the caller stops using the pool before calling Close. ErrClosed exists to
// surface that bug deterministically rather than silently allocate an
// untracked adapter.
var ErrClosed = errors.New("pool: closed")

type Adapter interface {
	Healthy() bool
	Close() error
	Acquire() error
	Release()
}

type Factory[K comparable, A Adapter] interface {
	New(K) (A, error)
}

type Pool[K comparable, A Adapter] interface {
	// Acquire returns an adapter for the given key, creating one via the
	// Factory if no healthy adapter is already cached.
	//
	// Calling Acquire after Close is a programming error: the contract is
	// that the caller stops using the pool before Close runs. As a safety
	// net for buggy callers, Acquire returns ErrClosed instead of silently
	// allocating an adapter that nothing will ever close.
	Acquire(key K) (A, error)
	// Close closes every adapter currently held by the pool.
	//
	// The caller is responsible for ensuring no goroutine is using the pool
	// concurrently with or after Close. Close is idempotent — second and
	// subsequent calls are no-ops. After Close, Acquire returns ErrClosed.
	Close() error
}

func New[K comparable, A Adapter](factory Factory[K, A]) Pool[K, A] {
	return &core[K, A]{pool: make(map[K][]A), factory: factory}
}

type core[K comparable, A Adapter] struct {
	factory Factory[K, A]
	pool    map[K][]A
	mu      sync.Mutex
}

func (p *core[K, A]) Acquire(key K) (a A, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pool == nil {
		return a, ErrClosed
	}
	if adapters, ok := p.pool[key]; ok {
		for _, adapter := range adapters {
			if adapter.Healthy() {
				return adapter, adapter.Acquire()
			}
		}
	}
	return p.new(key)
}

func (p *core[K, A]) Close() error {
	p.mu.Lock()
	adapters := p.pool
	p.pool = nil
	p.mu.Unlock()

	// Adapter teardowns happen outside the lock so a slow Close (e.g. grpc
	// client connection shutdown waiting on its internal WaitGroup) does not
	// block concurrent callers and cannot deadlock if an adapter's Close
	// path ever calls back into the pool.
	var err error
	for _, group := range adapters {
		for _, adapter := range group {
			err = errors.Combine(err, adapter.Close())
		}
	}
	return err
}

func (p *core[K, A]) new(key K) (a A, err error) {
	a, err = p.factory.New(key)
	if err != nil {
		return a, err
	}
	p.pool[key] = append(p.pool[key], a)
	return a, nil
}
