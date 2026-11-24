// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lifetime

import (
	"sync"

	"github.com/synnaxlabs/x/errors"
)

// Guard orders the Run and Close of a component. Run calls Begin when it starts and End
// when its goroutines exit. Close calls Close first and closes nothing when it fails.
// A component runs at most once, and never after it closes.
//
// The zero value is ready to use. Guard is safe for concurrent use.
type Guard struct {
	mu      sync.Mutex
	started bool
	running bool
	closed  bool
}

// Begin marks the component running. It returns an error when Begin was called before,
// or when Close was.
func (g *Guard) Begin() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return errors.New("run called after close")
	}
	if g.started {
		return errors.New("run called twice")
	}
	g.started, g.running = true, true
	return nil
}

// End marks the component stopped. Call it after the goroutines of Run exit. End
// panics when the component is not running.
func (g *Guard) End() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.running {
		panic("lifetime: End called without Begin")
	}
	g.running = false
}

// Close marks the component closed, so that a later Begin fails. It returns an error
// while the component runs, and then marks nothing. first is true only for the first
// Close that succeeds, so the caller releases its resources once.
func (g *Guard) Close() (first bool, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.running {
		return false, errors.New("close called while Run runs")
	}
	first = !g.closed
	g.closed = true
	return first, nil
}
