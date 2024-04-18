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
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"slices"
	"sync"
)

type (
	// State is the control State of a gate over a channel bound entity.
	State = control.State[core.ChannelKey]
	// Transfer is a transfer of control over a channel bound entity.
	Transfer = control.Transfer[core.ChannelKey]
)

// Entity represents some entity that can be controlled by a Gate. An Entity must have
// a ChannelKey that represents the resource that is being controlled.
type Entity interface {
	// ChannelKey returns the key of the channel that is being controlled.
	ChannelKey() core.ChannelKey
}

// Gate controls access to an entity for a given region of time.
type Gate[E Entity] struct {
	GateConfig
	r           *region[E]
	position    int64
	concurrency control.Concurrency
}

func (g *Gate[E]) State() *State {
	return &State{
		Subject:   g.Subject,
		Resource:  g.r.entity.ChannelKey(),
		Authority: g.Authority,
	}
}

// Authorize authorizes the gates access to the entity. If another gate has precedence,
// Authorize will return false.
func (g *Gate[E]) Authorize() (e E, ok bool) {
	g.r.RLock()
	// In the case of exclusive concurrency, we only need to check if the gate is the
	// current gate.
	if g.concurrency == control.Exclusive {
		ok = g.r.curr == g
	} else {
		// In the case of shared concurrency, we need to check if the gate has equal to
		// or higher authority than the current gate.
		ok = g.Authority >= g.r.curr.Authority
	}
	g.r.RUnlock()
	if !ok {
		return e, false
	}
	return g.r.entity, ok
}

// Release releases the gate's access to the entity. If the gate is the last gate in
// region (i.e. transfer.IsRelease() == true), the entity will be returned. Otherwise,
// the zero value of the entity will be returned.
func (g *Gate[E]) Release() (entity E, transfer Transfer) { return g.r.release(g) }

// SetAuthority changes the gate's authority, returning any transfer of control that
// may have occurred as a result.
func (g *Gate[E]) SetAuthority(auth control.Authority) Transfer {
	return g.r.update(g, auth)
}

type region[E Entity] struct {
	sync.RWMutex
	timeRange  telem.TimeRange
	entity     E
	counter    int64
	curr       *Gate[E]
	gates      map[*Gate[E]]struct{}
	controller *Controller[E]
}

// open opens a new gate on the region with the given config.
func (r *region[E]) open(c GateConfig, con control.Concurrency) (*Gate[E], Transfer, error) {
	r.Lock()
	g := &Gate[E]{
		r:           r,
		GateConfig:  c,
		position:    r.counter,
		concurrency: con,
	}
	t, err := r.unprotectedOpen(g)
	r.Unlock()
	return g, t, err
}

// release a gate from the region.
func (r *region[E]) release(g *Gate[E]) (e E, transfer Transfer) {
	r.Lock()
	e, transfer = r.unprotectedRelease(g)
	r.Unlock()
	return
}

// update a gate's authority.
func (r *region[E]) update(g *Gate[E], auth control.Authority) Transfer {
	r.Lock()
	t := r.unprotectedUpdate(g, auth)
	r.Unlock()
	return t
}

func (r *region[E]) unprotectedUpdate(
	g *Gate[E],
	auth control.Authority,
) (t Transfer) {
	prevAuth := g.Authority
	g.Authority = auth

	// Gate is in control, should it not be?
	if g == r.curr {
		t.From = g.State()
		t.From.Authority = prevAuth
		for og := range r.gates {
			var (
				isGate     = og == g
				higherAuth = og.Authority > g.Authority
				betterPos  = og.Authority == g.Authority && og.position < g.position
			)
			if !isGate && (higherAuth || betterPos) {
				r.curr = og
				t.From = g.State()
				t.To = og.State()
				return t
			}
		}
		// No transfer happened, gate remains in control.
		t.To = g.State()
		return t
	}

	// Gate is not in control, should it be?
	higherAuth := g.Authority > r.curr.Authority
	betterPos := g.Authority == r.curr.Authority && g.position < r.curr.position
	if higherAuth || betterPos {
		t.From = r.curr.State()
		r.curr = g
		t.To = g.State()
		return t
	}
	return
}

// unprotectedRelease releases a gate from the region without locking.
// If the gate is the last gate in the region, the region will be removed from
// the controller.
func (r *region[E]) unprotectedRelease(g *Gate[E]) (e E, t Transfer) {
	delete(r.gates, g)

	if len(r.gates) == 0 {
		r.controller.remove(r)
		t.From = g.State()
		return r.entity, t
	}

	if len(r.gates) == 1 {
		t.From = r.curr.State()
		for k := range r.gates {
			r.curr = k
			t.To = k.State()
		}
		return r.entity, t
	}

	if g == r.curr {
		t.From = r.curr.State()
		r.curr = nil
		for og := range r.gates {
			// Three cases here: no one is in control, provided gate has higher authority,
			// provided gate has equal authority and a higher position.
			if r.curr == nil || og.Authority > r.curr.Authority || (og.Authority == r.curr.Authority && og.position < r.curr.position) {
				r.curr = og
				t.To = og.State()
			}
		}
	}
	return r.entity, t
}

func (r *region[E]) unprotectedOpen(g *Gate[E]) (t Transfer, err error) {
	// Check if any gates have the same subject key.
	for og := range r.gates {
		if og.Subject.Key == g.Subject.Key {
			err = errors.Wrapf(validate.Error, "[controller] - gate with subject key %s already exists", g.Subject.Key)
			return
		}
	}

	if r.curr == nil || g.Authority > r.curr.Authority {
		if r.curr != nil {
			t.From = r.curr.State()
		}
		r.curr = g
		t.To = g.State()
	}
	r.gates[g] = struct{}{}
	r.counter++
	return
}

type Controller[E Entity] struct {
	mu          sync.RWMutex
	regions     []*region[E]
	concurrency control.Concurrency
}

func New[E Entity](c control.Concurrency) *Controller[E] {
	return &Controller[E]{
		regions:     make([]*region[E], 0),
		concurrency: c,
	}
}

// GateConfig is the configuration for opening a gate.
type GateConfig struct {
	// TimeRange sets the time range for the gate. Any subsequent calls to OpenGate
	// with overlapping time ranges will bind themselves to the same control region.
	TimeRange telem.TimeRange
	// Authority sets the authority of the gate over the entity. For a complete
	// discussion of authority, see the package level documentation.
	Authority control.Authority
	// Subject sets the identity of the gate, and is used to track changes in control
	// within the db.
	Subject control.Subject
}

var (
	_ config.Config[GateConfig] = GateConfig{}
	// DefaultGateConfig is the default configuration for opening a Gate.
	DefaultGateConfig = GateConfig{}
)

// Validate implements config.Properties.
func (c GateConfig) Validate() error {
	v := validate.New("gate config")
	validate.NotEmptyString(v, "subject.key", c.Subject.Key)
	validate.NonZeroable(v, "TimeRange", c.TimeRange)
	return v.Error()
}

// Override implements config.Properties.
func (c GateConfig) Override(other GateConfig) GateConfig {
	c.Authority = override.Numeric(c.Authority, other.Authority)
	c.Subject.Key = override.String(c.Subject.Key, other.Subject.Key)
	c.Subject.Name = override.String(c.Subject.Name, other.Subject.Name)
	c.TimeRange.Start = override.Numeric(c.TimeRange.Start, other.TimeRange.Start)
	c.TimeRange.End = override.Numeric(c.TimeRange.End, other.TimeRange.End)
	return other
}

// LeadingState returns the current control State of the leading region in the
// controller. Returns nil if no regions are under control.
func (c *Controller[E]) LeadingState() *State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.regions) == 0 {
		return nil
	}
	first := c.regions[0]
	if len(first.gates) == 0 {
		return nil
	}
	return first.curr.State()
}

// OpenAbsoluteGateIfUncontrolled opens a region and an absolute gate on a timerange if
// it is not under control of another region otherwise. Otherwise, it returns an error
func (c *Controller[E]) OpenAbsoluteGateIfUncontrolled(tr telem.TimeRange, s control.Subject, callback func() (E, error)) (g *Gate[E], t Transfer, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var (
		gateCfg = GateConfig{
			TimeRange: tr,
			Authority: control.Absolute,
			Subject:   s,
		}
		exists = false
	)

	for _, r := range c.regions {
		// Check if there is an existing region that overlaps with that time range,
		// if there is, then that means a gate is already in control of it,
		// therefore this method should not create an absolute gate.
		if r.timeRange.OverlapsWith(tr) {
			if exists {
				return nil, t, errors.Newf("[controller] - encountered multiple control regions for time range %s", tr)
			}

			r.Lock()
			if r.curr != nil {
				r.Unlock()
				return nil, t, errors.Newf("[controller] - region already being controlled")
			}

			g = &Gate[E]{
				r:           r,
				GateConfig:  gateCfg,
				position:    r.counter,
				concurrency: c.concurrency,
			}

			t, err = r.unprotectedOpen(g)
			if err != nil {
				return nil, t, err
			}
			r.gates[g] = struct{}{}

			r.Unlock()
			exists = true
		}
	}
	if !exists {
		e, err := callback()
		if err != nil {
			return g, t, err
		}
		r := c.insertNewRegion(tr, e)
		g, t, err = r.open(gateCfg, c.concurrency)
		r.gates[g] = struct{}{}
	}
	return
}

// OpenGateAndMaybeRegister checks whether a region exists in the requested timerange,
// and creates a new one if one does. Otherwise, it registers that region and opens a
// new gate with absolute authority.
func (c *Controller[E]) OpenGateAndMaybeRegister(cfg GateConfig, callback func() (E, error)) (g *Gate[E], t Transfer, err error) {
	cfg, err = config.New(DefaultGateConfig, cfg)
	var exists bool
	if err != nil {
		return nil, t, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, r := range c.regions {
		// Check if there is an existing region that overlaps with that timerange.
		if r.timeRange.OverlapsWith(cfg.TimeRange) {
			// v1 optimization: one writer can only overlap with one region at any given time.
			if exists {
				return nil, t, errors.Newf("[controller] - encountered multiple control regions for time range %s", cfg.TimeRange)
			}
			// If there is an existing region, we open a new gate on that region.
			g, t, err = r.open(cfg, c.concurrency)
			if err != nil {
				return nil, t, err
			}
			r.gates[g] = struct{}{}
			exists = true
		}
	}

	if !exists {
		e, err := callback()
		if err != nil {
			return g, t, err
		}
		r := c.insertNewRegion(cfg.TimeRange, e)
		g, t, err = r.open(cfg, c.concurrency)
		r.gates[g] = struct{}{}
	}
	return
}

func (c *Controller[E]) Register(
	t telem.TimeRange,
	entity E,
) error {
	c.mu.Lock()
	err := c.register(t, entity)
	c.mu.Unlock()
	return err
}

func (c *Controller[E]) register(
	t telem.TimeRange,
	entity E,
) error {
	for _, r := range c.regions {
		if r.timeRange.OverlapsWith(t) {
			return errors.Newf("entity already registered for time range %s", t)
		}
	}
	c.insertNewRegion(t, entity)
	return nil
}

func (c *Controller[E]) insertNewRegion(
	t telem.TimeRange,
	entity E,
) *region[E] {
	r := &region[E]{
		entity:     entity,
		gates:      make(map[*Gate[E]]struct{}),
		timeRange:  t,
		controller: c,
	}
	pos, _ := slices.BinarySearchFunc(c.regions, r, func(a *region[E], b *region[E]) int {
		return int(a.timeRange.Start - b.timeRange.Start)
	})
	c.regions = slices.Insert(c.regions, pos, r)
	return r
}

func (c *Controller[E]) remove(r *region[E]) {
	c.mu.Lock()
	for i, reg := range c.regions {
		if reg == r {
			c.regions = append(c.regions[:i], c.regions[i+1:]...)
			break
		}
	}
	c.mu.Unlock()
}

func Unauthorized(name string, ch core.ChannelKey) error {
	return errors.Wrapf(control.Unauthorized, "writer %s does not have control authority over channel %s", name, ch)
}
