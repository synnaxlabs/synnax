// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package client

import (
	"context"
	"sync"

	"github.com/synnaxlabs/x/errors"
)

// Pool manages shared Sift client connections.
type Pool struct {
	mu      sync.RWMutex
	clients map[string]Client
	factory Factory
}

// NewPool creates a new client pool with the given factory.
func NewPool(factory Factory) *Pool {
	return &Pool{clients: make(map[string]Client), factory: factory}
}

// Get retrieves or creates a client for the given URI and API key.
func (p *Pool) Get(ctx context.Context, uri, apiKey string) (Client, error) {

	p.mu.RLock()
	if client, ok := p.clients[uri]; ok {
		defer p.mu.RUnlock()
		return client, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock, since there may have been a race
	// condition between releasing the read lock and acquiring the write lock.
	if client, ok := p.clients[uri]; ok {
		return client, nil
	}

	client, err := p.factory(ctx, uri, apiKey)
	if err != nil {
		return nil, err
	}
	p.clients[uri] = client
	return client, nil
}

// Close closes all connections in the pool.
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	a := errors.NewCatcher(errors.WithAggregation())
	for _, client := range p.clients {
		a.Exec(client.Close)
	}
	clear(p.clients)
	return a.Error()
}
