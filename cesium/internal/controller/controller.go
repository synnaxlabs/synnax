// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package controller

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	"sync"
)

type Gate[E any] struct {
	r           *region[E]
	concurrency control.Concurrency
	position    int
	authority   control.Authority
}

func (g *Gate[E]) Authorize() (e E, ok bool) {
	g.r.RLock()
	if g.concurrency == control.Exclusive {
		ok = g.r.curr == g
	} else {
		ok = g.authority >= g.r.curr.authority
	}
	g.r.RUnlock()
	if !ok {
		return e, false
	}
	return g.r.entity, ok
}

func (g *Gate[E]) Release() (entity E, regionReleased bool) {
	return g.r.release(g)
}

type region[E any] struct {
	sync.RWMutex
	timeRange   telem.TimeRange
	entity      E
	counter     int
	curr        *Gate[E]
	authorities map[*Gate[E]]struct{}
	controller  *Controller[E]
}

func (r *region[E]) open(auth control.Authority, conc control.Concurrency) *Gate[E] {
	r.Lock()
	g := &Gate[E]{r: r, position: r.counter, authority: auth, concurrency: conc}
	r.authorities[g] = struct{}{}
	if r.curr == nil || g.authority > r.curr.authority {
		r.curr = g
	}
	r.counter++
	r.Unlock()
	return g
}

func (r *region[E]) release(g *Gate[E]) (E, bool) {
	r.Lock()
	delete(r.authorities, g)
	if len(r.authorities) == 0 {
		r.Unlock()
		r.controller.remove(r)
		return r.entity, true
	}
	if g == r.curr {
		r.curr = nil
		for g := range r.authorities {
			if r.curr == nil || g.authority > r.curr.authority || (g.authority == r.curr.authority && g.position > r.curr.position) {
				r.curr = g
			}
		}
	}
	r.Unlock()
	return r.entity, false
}

type Controller[E any] struct {
	mu          sync.Mutex
	regions     map[telem.TimeRange]*region[E]
	concurrency control.Concurrency
}

func New[E any](conc control.Concurrency) *Controller[E] {
	return &Controller[E]{regions: make(map[telem.TimeRange]*region[E]), concurrency: conc}
}

func (c *Controller[E]) OpenGate(t telem.TimeRange, auth control.Authority) (g *Gate[E], exists bool) {
	c.mu.Lock()
	for _, r := range c.regions {
		if r.timeRange.OverlapsWith(t) {
			g = r.open(auth, c.concurrency)
			r.authorities[g] = struct{}{}
			c.mu.Unlock()
			return g, true
		}
	}
	c.mu.Unlock()
	return nil, false
}

func (c *Controller[E]) Register(
	t telem.TimeRange,
	entity E,
) error {
	c.mu.Lock()
	for _, r := range c.regions {
		if r.timeRange.OverlapsWith(t) {
			c.mu.Unlock()
			return errors.Newf("entity already registered for time range %s", t)
		}
	}
	c.regions[t] = &region[E]{entity: entity, authorities: make(map[*Gate[E]]struct{}), timeRange: t, controller: c}
	c.mu.Unlock()
	return nil
}

func (c *Controller[E]) RegisterAndOpenGate(
	t telem.TimeRange,
	auth control.Authority,
	entity E,
) *Gate[E] {
	c.mu.Lock()
	r := &region[E]{entity: entity, authorities: make(map[*Gate[E]]struct{}, 1), timeRange: t, controller: c}
	g := r.open(auth, c.concurrency)
	r.authorities[g] = struct{}{}
	c.regions[t] = r
	c.mu.Unlock()
	return g
}

func (c *Controller[E]) remove(r *region[E]) {
	c.mu.Lock()
	delete(c.regions, r.timeRange)
	c.mu.Unlock()
}

func Unauthorized(ch core.ChannelKey) error {
	return errors.Wrapf(control.Unauthorized, "writer does not have control authority over channel %s", ch)
}
