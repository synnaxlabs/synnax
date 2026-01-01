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
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
)

// Gate controls access to a resource for a given region of time.
type Gate[R Resource] struct {
	// subject is the subject attempting to control the resource that the gate
	// is guarding access to.
	subject control.Subject
	// authority is the authority that the subject has over the resource.
	authority control.Authority
	// region is the control region in time for the resource
	region *region[R]
	// position is the number of gates that had been opened in the region before
	// this gate. Gates with a lower position yet equal authority take precedence
	// over this gate. This position is constant for the lifetime of the gate.
	position uint
}

// Subject returns information about the subject controlling this gate.
func (g *Gate[R]) Subject() control.Subject { return g.subject }

// Authority returns the control authority for this gate.
func (g *Gate[R]) Authority() control.Authority { return g.authority }

// State returns the current control State of the gate.
func (g *Gate[R]) state() *State {
	return &State{
		Subject:   g.subject,
		Resource:  g.region.resource.ChannelKey(),
		Authority: g.authority,
	}
}

// PeekResource returns the resource controlled by the gate. The resource is NOT valid
// for modification or use.
func (g *Gate[R]) PeekResource() R {
	g.region.RLock()
	defer g.region.RUnlock()
	return g.region.resource
}

// Authorize authorizes the gate's access to the resource. If another gate has precedence,
// Authorize will return a control.ErrUnauthorized error, and the zero value for the resource.
// If the gate has control over the resource, returns the resource and a nil error.
func (g *Gate[R]) Authorize() (r R, err error) {
	g.region.RLock()
	defer g.region.RUnlock()
	if g.region == nil || g.region.curr == nil {
		return r, errors.Wrapf(
			control.ErrUnauthorized,
			"%s has no control authority - gate was already released",
			g.Subject(),
		)
	}
	// In the case of exclusive concurrency, we only need to check if the gate is the
	// current gate.
	if g.region.controller.Concurrency == control.Exclusive {
		if g.region.curr == g {
			return g.region.resource, nil
		}
	} else if g.authority >= g.region.curr.authority {
		return g.region.resource, nil
	}
	return r, errors.Wrapf(
		control.ErrUnauthorized,
		"%s has no control authority - it is currently held by %s",
		g.Subject(),
		g.region.curr.Subject(),
	)
}

// Release releases the gate's access to the resource. If the gate is the last gate in
// a region, i.e., transfer.IsRelease() == true, the resource will be returned. Otherwise,
// the zero value of the resource will be returned.
func (g *Gate[R]) Release() (resource R, transfer Transfer) { return g.region.release(g) }

// SetAuthority changes the gate's authority, returning any transfer of control that
// may have occurred as a result.
func (g *Gate[R]) SetAuthority(auth control.Authority) Transfer {
	return g.region.update(g, auth)
}
