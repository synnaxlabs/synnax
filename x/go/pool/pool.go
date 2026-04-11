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

// ErrClosed is returned by Acquire after Close has been called. It signals
// callers that they must stop using this pool.
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
	Acquire(key K) (A, error)
	// Close closes every adapter held by the pool and marks the pool as
	// closed. Subsequent Acquire calls return ErrClosed. Close is safe to
	// call multiple times — the second call is a no-op.
	Close() error
}

func New[K comparable, A Adapter](factory Factory[K, A]) Pool[K, A] {
	return &core[K, A]{pool: make(map[K][]A), factory: factory}
}

type core[K comparable, A Adapter] struct {
	factory Factory[K, A]
	pool    map[K][]A
	mu      sync.RWMutex
	closed  bool
}

func (p *core[K, A]) Acquire(key K) (a A, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
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
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	var err error
	for _, adapters := range p.pool {
		for _, adapter := range adapters {
			err = errors.Combine(err, adapter.Close())
		}
	}
	p.pool = make(map[K][]A)
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
