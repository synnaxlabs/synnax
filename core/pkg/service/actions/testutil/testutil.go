// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides shared helpers for testing action-based services
// (line plots, tables, schematics, logs). It is intended to be dot-imported into
// a service's *_test package.
package testutil

import (
	"context"
	"sync"

	"github.com/synnaxlabs/synnax/pkg/service/actions"
)

// Recorder collects the Scoped action notifications a service emits through its
// action observer so tests can assert on them. The zero value is ready to use,
// and Recorder is safe for concurrent use. Pass Record directly to
// Service.OnAction:
//
//	rec := &Recorder[log.Key, log.Action]{}
//	DeferCleanup(svc.OnAction(rec.Record))
type Recorder[K comparable, A any] struct {
	mu      sync.Mutex
	scopeds []actions.Scoped[K, A]
}

// Record appends the given scoped action. Its signature matches the handler
// expected by Service.OnAction.
func (r *Recorder[K, A]) Record(_ context.Context, sa actions.Scoped[K, A]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scopeds = append(r.scopeds, sa)
}

// Snapshot returns a copy of the scoped actions recorded so far in the order they
// were received.
func (r *Recorder[K, A]) Snapshot() []actions.Scoped[K, A] {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]actions.Scoped[K, A], len(r.scopeds))
	copy(out, r.scopeds)
	return out
}
