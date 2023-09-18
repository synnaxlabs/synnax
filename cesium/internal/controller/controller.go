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
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/reflect"
	"github.com/synnaxlabs/x/telem"
	"sync"
)

type Gate[E any] struct {
	Config
	r           *region[E]
	position    int
	concurrency control.Concurrency
	hasDigests  bool
}

type Digest struct {
	Name        string
	Authority   control.Authority
	Concurrency control.Concurrency
}

func (g *Gate[E]) Authorize() (e E, ok bool) {
	g.r.RLock()
	if g.concurrency == control.Exclusive {
		ok = g.r.curr == g
	} else {
		ok = g.Authority >= g.r.curr.Authority
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

func (g *Gate[E]) SetAuthority(auth control.Authority) {
	g.r.update(g, auth)
}

type region[E any] struct {
	sync.RWMutex
	timeRange  telem.TimeRange
	entity     E
	counter    int
	curr       *Gate[E]
	gates      map[*Gate[E]]struct{}
	controller *Controller[E]
}

func (r *region[E]) open(c Config, con control.Concurrency) *Gate[E] {
	r.Lock()
	g := &Gate[E]{
		r:           r,
		Config:      c,
		position:    r.counter,
		concurrency: con,
		hasDigests:  !reflect.IsNil(c.Digests),
	}
	r.unprotectedOpen(g)
	r.Unlock()
	return g
}

func (r *region[E]) release(g *Gate[E]) (e E, regionReleased bool) {
	r.Lock()
	if g.hasDigests {
		g.Digests.Close()
		g.Digests = nil
	}
	e, regionReleased = r.unprotectedRelease(g)
	r.Unlock()
	return e, regionReleased
}

func (r *region[E]) update(g *Gate[E], auth control.Authority) {
	r.Lock()
	r.unprotectedRelease(g)
	g.Authority = auth
	r.unprotectedOpen(g)
	r.Unlock()
}

func (r *region[E]) unprotectedRelease(g *Gate[E]) (E, bool) {
	delete(r.gates, g)
	if len(r.gates) == 0 {
		r.controller.remove(r)
		return r.entity, true
	}
	if g == r.curr {
		r.curr = nil
		for g := range r.gates {
			// Three cases here: no one is in control, provided gate has higher authority,
			// provided gate has equal authority and a higher position.
			if r.curr == nil || g.Authority > r.curr.Authority || (g.Authority == r.curr.Authority && g.position > r.curr.position) {
				r.unprotectedSetCurr(g)
			}
		}
	}
	return r.entity, false
}

func (r *region[E]) unprotectedSetCurr(g *Gate[E]) {
	r.curr = g
	var dig Digest
	if g == nil {
		dig = Digest{Name: "None", Concurrency: r.controller.concurrency}
	} else {
		dig = Digest{Name: g.Name, Authority: g.Authority, Concurrency: g.concurrency}
	}
	for og := range r.gates {
		if og.hasDigests {
			og.Digests.Inlet() <- dig
		}
	}
}

func (r *region[E]) unprotectedOpen(g *Gate[E]) {
	r.gates[g] = struct{}{}
	if r.curr == nil || g.Authority > r.curr.Authority {
		if g.hasDigests {
			g.Digests.Acquire(1)
		}
		r.unprotectedSetCurr(g)
	}
	r.counter++
}

type Controller[E any] struct {
	mu          sync.Mutex
	regions     map[telem.TimeRange]*region[E]
	concurrency control.Concurrency
}

func New[E any](conc control.Concurrency) *Controller[E] {
	return &Controller[E]{regions: make(map[telem.TimeRange]*region[E]), concurrency: conc}
}

type Config struct {
	TimeRange telem.TimeRange
	Authority control.Authority
	Digests   confluence.Inlet[Digest]
	Name      string
}

func (c *Controller[E]) OpenGate(cfg Config) (g *Gate[E], exists bool) {
	c.mu.Lock()
	for _, r := range c.regions {
		if r.timeRange.OverlapsWith(cfg.TimeRange) {
			g = r.open(cfg, c.concurrency)
			r.gates[g] = struct{}{}
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
	c.regions[t] = &region[E]{entity: entity, gates: make(map[*Gate[E]]struct{}), timeRange: t, controller: c}
	c.mu.Unlock()
	return nil
}

func (c *Controller[E]) RegisterAndOpenGate(
	cfg Config,
	entity E,
) *Gate[E] {
	c.mu.Lock()
	r := &region[E]{
		entity:     entity,
		gates:      make(map[*Gate[E]]struct{}, 1),
		timeRange:  cfg.TimeRange,
		controller: c,
	}
	g := r.open(cfg, c.concurrency)
	r.gates[g] = struct{}{}
	c.regions[cfg.TimeRange] = r
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
