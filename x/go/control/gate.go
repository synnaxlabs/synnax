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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// Gate controls access to a resource for a given region of time.
type Gate[K comparable, R Resource[K]] struct {
	// region is the control region in time for the resource
	region *region[K, R]
	// subject is the subject attempting to control the resource that the gate
	// is guarding access to.
	subject Subject
	// position is the number of gates that had been opened in the region before
	// this gate. Gates with a lower position yet equal authority take precedence
	// over this gate. This position is constant for the lifetime of the gate.
	position uint
	// authority is the authority that the subject has over the resource.
	authority Authority
}

// Subject returns information about the subject controlling this gate.
func (g *Gate[K, R]) Subject() Subject { return g.subject }

// Authority returns the control authority for this gate.
func (g *Gate[K, R]) Authority() Authority { return g.authority }

// State returns the current control State of the gate.
func (g *Gate[K, R]) state() *State[K] {
	return &State[K]{
		Subject:   g.subject,
		Resource:  g.region.resource.Key(),
		Authority: g.authority,
	}
}

// PeekResource returns the resource controlled by the gate. The resource is NOT valid
// for modification or use.
func (g *Gate[K, R]) PeekResource() R {
	g.region.RLock()
	defer g.region.RUnlock()
	return g.region.resource
}

// Authorize calls f with the resource if the gate has control over it, returning the
// error from f. If another gate has precedence, Authorize returns an ErrUnauthorized
// error without calling f. Calls to f are serialized across all gates in the region, so
// f has exclusive access to the resource even if control transfers while f runs. f must
// not call Authorize on a gate in the same region.
func (g *Gate[K, R]) Authorize(f func(R) error) error {
	// Fail without waiting on use, so a gate without control never delays the gate
	// that has it.
	if _, err := g.authorize(); err != nil {
		return err
	}
	g.region.use.Lock()
	defer g.region.use.Unlock()
	// Control can transfer while this gate waits on use.
	r, err := g.authorize()
	if err != nil {
		return err
	}
	return f(r)
}

func (g *Gate[K, R]) authorize() (r R, err error) {
	g.region.RLock()
	defer g.region.RUnlock()
	if g.region == nil || g.region.curr == nil {
		return r, errors.Wrapf(
			ErrUnauthorized,
			"%s has no control authority - gate was already released",
			g.Subject(),
		)
	}
	// In the case of exclusive concurrency, we only need to check if the gate is the
	// current gate.
	if g.region.controller.Concurrency == ConcurrencyExclusive {
		if g.region.curr == g {
			return g.region.resource, nil
		}
	} else if g.authority >= g.region.curr.authority {
		return g.region.resource, nil
	}
	return r, errors.Wrapf(
		ErrUnauthorized,
		"%s has no control authority - it is currently held by %s",
		g.Subject(),
		g.region.curr.Subject(),
	)
}

// MoveStart moves the start of the gate's region to start, which frees any time before
// it. It fails when the region would reach another region. A bounded region only moves
// back, and a start at or after the end of the region does nothing.
func (g *Gate[K, R]) MoveStart(start telem.TimeStamp) error {
	return g.region.controller.moveStart(g.region, start)
}

// Reach moves the end of the gate's region forward to end. It does nothing when the
// region already holds end, and fails when the region would reach another region.
func (g *Gate[K, R]) Reach(end telem.TimeStamp) error {
	return g.region.controller.reach(g.region, end)
}

// Release releases the gate's access to the resource. If the gate is the last gate in a
// region, i.e., transfer.IsRelease() == true, the resource will be returned. Otherwise,
// the zero value of the resource will be returned.
func (g *Gate[K, R]) Release() (resource R, transfer Transfer[K]) {
	return g.region.release(g)
}

// SetAuthority changes the gate's authority, returning any transfer of control that
// may have occurred as a result.
func (g *Gate[K, R]) SetAuthority(auth Authority) Transfer[K] {
	return g.region.update(g, auth)
}
