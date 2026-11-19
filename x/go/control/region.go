// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control

import (
	"sync"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type region[K comparable, R Resource[K]] struct {
	// resource is the resource being controlled by the region
	// [not safe for unprotected concurrent access by external callers, but safe to
	// call static properties like Channel() concurrently]
	resource R
	// curr is the gate currently in control of the region.
	// [not safe for unprotected concurrent access]
	curr *Gate[K, R]
	// gates are the gates vying for control of the region. curr is one of the gates
	// in this map.
	// [not safe for unprotected concurrent access]
	gates set.Set[*Gate[K, R]]
	// controller is the parent controller.
	// [not safe for unprotected concurrent access]
	controller *Controller[K, R]
	// timeRange tracks the time period that the region controls. Gates that access
	// independent regions can access a resource at the same time.
	// [not safe for unprotected concurrent access]
	timeRange telem.TimeRange
	// bounded is true when the region holds a single bounded gate. It does not change
	// after the region is created.
	bounded bool
	// counter tracks the total number of gates opened in the region. This counter
	// does not get decremented.
	// [not safe for unprotected concurrent access]
	counter uint
	// opening is non-nil from when the region is created until its resource opens or
	// fails to, and closes at that point. A region that is opening has no gates and no
	// resource. Guarded by controller.mu.
	opening chan struct{}
	// use serializes access to resource by authorized gates.
	use sync.Mutex
	// RWMutex locks access to the region's state. Only specified fields in a region are
	// safe for concurrent access without locking the mutex.
	sync.RWMutex
}

// open opens a new gate on the region with the given config.
func (r *region[K, R]) open(
	cfg GateConfig[K, R],
) (g *Gate[K, R], t Transfer[K], err error) {
	r.Lock()
	defer r.Unlock()
	if *cfg.ErrIfControlled && r.curr != nil {
		err = errors.Wrapf(
			ErrUnauthorized,
			"cannot access time range %v because writer %v holds it",
			cfg.TimeRange.Intersection(r.timeRange),
			r.curr.Subject(),
		)
		return g, t, err
	}

	// Check if any gates have the same subject key.
	for existingG := range r.gates {
		if existingG.subject.Key == cfg.Subject.Key {
			return g, t, errors.Wrapf(
				validate.ErrValidation,
				"control subject %s is already registered in the region",
				cfg.Subject,
			)
		}
	}

	g = &Gate[K, R]{
		region:    r,
		subject:   cfg.Subject,
		authority: cfg.Authority,
		position:  r.counter,
	}

	// If no one is in control or this gate has a higher authority, take control.
	if r.curr == nil || g.authority > r.curr.authority {
		if r.curr != nil {
			t.From = r.curr.state()
		}
		r.curr = g
		t.To = g.state()
	} else if *cfg.ErrOnUnauthorizedOpen && (r.controller.Concurrency != ConcurrencyShared || g.authority != r.curr.authority) {
		err = errors.Wrapf(
			ErrUnauthorized,
			"%s has no control authority - it is currently held by %s",
			g.Subject(),
			r.curr.Subject(),
		)
		g = nil
		return g, t, err
	}
	r.timeRange = r.timeRange.Union(cfg.TimeRange)
	r.gates.Add(g)
	r.counter++
	return g, t, nil
}

// state returns the control State of the gate in control, or nil when the region is
// bounded or still opening.
func (r *region[K, R]) state() *State[K] {
	r.RLock()
	defer r.RUnlock()
	if r.bounded || r.curr == nil {
		return nil
	}
	return r.curr.state()
}

// overlapError returns the error for a write to tr that reaches the region.
func (r *region[K, R]) overlapError(tr telem.TimeRange) error {
	r.RLock()
	defer r.RUnlock()
	holder := "a writer that is opening"
	if r.curr != nil {
		holder = "writer " + r.curr.Subject().String()
	}
	return errors.Wrapf(
		validate.ErrValidation,
		"cannot write to time range %v because %s holds it",
		tr.Intersection(r.timeRange),
		holder,
	)
}

func (r *region[K, R]) shouldBeInControl(candidate *Gate[K, R]) bool {
	if r.curr == nil {
		return true
	}
	// Three cases here: no one is in control, provided-gate has higher authority,
	// a provided gate has equal authority and a higher position.
	higherAuth := candidate.authority > r.curr.authority
	betterPos := candidate.authority == r.curr.authority &&
		candidate.position < r.curr.position
	return higherAuth || betterPos
}

// release a gate from the region. Releasing the last gate removes the region from the
// controller in the same critical section, so no gate can join a region whose resource
// the caller is about to close.
func (r *region[K, R]) release(g *Gate[K, R]) (res R, transfer Transfer[K]) {
	r.controller.mu.Lock()
	defer r.controller.mu.Unlock()
	r.Lock()
	defer r.Unlock()
	r.gates.Remove(g)
	if r.curr != g {
		return res, transfer
	}
	r.curr = nil
	transfer.From = g.state()
	for candidate := range r.gates {
		if r.shouldBeInControl(candidate) {
			r.curr = candidate
			transfer.To = candidate.state()
		}
	}
	if transfer.IsRelease() {
		r.controller.unsafeRemove(r)
	}
	return r.resource, transfer
}

// update a gate's authority.
func (r *region[K, R]) update(g *Gate[K, R], auth Authority) (t Transfer[K]) {
	r.Lock()
	defer r.Unlock()
	if !r.gates.Contains(g) {
		return t
	}
	prevAuth := g.authority
	g.authority = auth

	// Gate is in control, should it not be?
	if g == r.curr {
		t.From = g.state()
		t.From.Authority = prevAuth
		// Iterate all gates to find the best candidate, not just the first match,
		// since map iteration order is non-deterministic.
		for existingGate := range r.gates {
			if r.shouldBeInControl(existingGate) {
				r.curr = existingGate
			}
		}
		if r.curr != g {
			t.To = r.curr.state()
			return t
		}
		// No transfer happened, gate remains in control.
		t.To = g.state()
		return t
	}

	if r.shouldBeInControl(g) {
		t.From = r.curr.state()
		r.curr = g
		t.To = g.state()
	}
	return t
}
