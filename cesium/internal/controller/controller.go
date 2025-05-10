// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package controller

import (
	"slices"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type (
	// State is the control State of a gate over a channel-bound resource.
	State = control.State[core.ChannelKey]
	// Transfer is a transfer of control over a channel-bound resource.
	Transfer = control.Transfer[core.ChannelKey]
)

// Resource represents some resource that can be controlled by a Gate. A Resource must have
// a ChannelKey that represents the resource that is being controlled.
type Resource interface {
	// ChannelKey returns the key of the channel that is being controlled.
	ChannelKey() core.ChannelKey
}

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
	// this gate. Gates with a higher position yet equal authority take precedence
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

// Authorized authorizes the gates' access to the resource. If another gate has precedence,
// Authorized will return false.
func (g *Gate[R]) Authorized() (e R, ok bool) {
	e, err := g.Authorize()
	return e, err == nil
}

// Authorize authorizes the gate's access to the resource. If another gate has precedence,
// Authorize will return a control.Unauthorized error, and the zero value for the resource.
// If the gate has control over the resource, returns the resource and a nil error.
func (g *Gate[R]) Authorize() (r R, err error) {
	g.region.RLock()
	defer g.region.RUnlock()
	// In the case of exclusive concurrency, we only need to check if the gate is the
	// current gate.
	var ok bool
	if g.region.controller.Concurrency == control.Exclusive {
		ok = g.region.curr == g
	} else {
		// In the case of shared concurrency, we need to check if the gate has equal to
		// or higher authority than the current gate.
		ok = g.authority >= g.region.curr.authority
	}
	if !ok {
		if g.region == nil || g.region.curr == nil {
			return r, errors.Wrapf(
				control.Unauthorized,
				"%s has no control authority - gate was already released",
				g.Subject(),
			)
		}
		return r, errors.Wrapf(
			control.Unauthorized,
			"%s has no control authority - it is currently held by %s",
			g.Subject(),
			g.region.curr.Subject(),
		)
	}
	return g.region.resource, nil
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

type region[R Resource] struct {
	// RWMutex locks access to the region's state. Only specified fields in a region are
	// safe for concurrent access without locking the mutex.
	sync.RWMutex
	// timeRange tracks the time period that the region controls. Gates that access
	// independent regions can access a resource at the same time.
	// [not safe for unprotected concurrent access]
	timeRange telem.TimeRange
	// resource is the resource being controlled by the region
	// [not safe for unprotected concurrent access by external callers, but safe to
	// call static properties like ChannelKey() concurrently]
	resource R
	// counter tracks the total number of gates opened in the region. This counter
	// does not get decremented.
	// [not safe for unprotected concurrent access]
	counter uint
	// curr is the gate currently in control of the region.
	// [not safe for unprotected concurrent access]
	curr *Gate[R]
	// gates are the gates vying for control of the region. curr is one of the gates
	// in this map.
	// [not safe for unprotected concurrent access]
	gates map[*Gate[R]]struct{}
	// controller is the parent controller.
	// [not safe for unprotected concurrent access]
	controller *Controller[R]
}

// open opens a new gate on the region with the given config.
func (r *region[R]) open(cfg GateConfig[R]) (g *Gate[R], t Transfer, err error) {
	r.Lock()
	defer r.Unlock()
	if *cfg.ErrIfControlled && r.curr != nil {
		err = errors.Wrapf(
			control.Unauthorized,
			"time range %v overlaps with a controlled region with bounds %v controlled by %v",
			cfg.TimeRange,
			r.timeRange,
			r.curr.Subject(),
		)
		return
	}

	g = &Gate[R]{
		region:    r,
		subject:   cfg.Subject,
		authority: cfg.Authority,
		position:  r.counter,
	}

	// Check if any gates have the same subject key.
	for existingG := range r.gates {
		if existingG.subject.Key == g.subject.Key {
			err = errors.Wrapf(
				validate.Error,
				"control subject %s is already registered in the region. Did you open two writers with the same key?",
				g.Subject(),
			)
			g = nil
			return
		}
	}

	// Expand the time range to include the new gate's time range.
	r.timeRange = r.timeRange.MaxUnion(cfg.TimeRange)

	// If no one is in control or this gate has a higher authority, take control.
	if r.curr == nil || g.authority > r.curr.authority {
		if r.curr != nil {
			t.From = r.curr.state()
		}
		r.curr = g
		t.To = g.state()
	} else if *cfg.ErrOnUnauthorizedOpen && !(r.controller.Concurrency == control.Shared && g.authority == r.curr.authority) {
		err = errors.Wrapf(
			control.Unauthorized,
			"%s has no control authority - it is currently held by %s",
			g.Subject(),
			r.curr.Subject(),
		)
		g = nil
		return
	}
	r.gates[g] = struct{}{}
	r.counter++
	return
}

// release a gate from the region.
func (r *region[R]) release(g *Gate[R]) (res R, transfer Transfer) {
	r.Lock()
	defer r.Unlock()
	delete(r.gates, g)
	if wasInControl := r.curr == g; !wasInControl {
		return
	}
	r.curr = nil
	transfer.From = g.state()
	for existingG := range r.gates {
		if r.curr == nil {
			r.curr = existingG
			transfer.To = existingG.state()
		} else {
			// Three cases here: no one is in control, provided-gate has higher authority,
			// a provided gate has equal authority and a higher position.
			higherAuth := existingG.authority > r.curr.authority
			betterPos := existingG.authority == r.curr.authority && existingG.position < r.curr.position
			if higherAuth || betterPos {
				r.curr = existingG
				transfer.To = existingG.state()
			}
		}
	}
	if transfer.IsRelease() {
		r.controller.remove(r)
	}
	return r.resource, transfer
}

// update a gate's authority.
func (r *region[R]) update(g *Gate[R], auth control.Authority) (t Transfer) {
	r.Lock()
	defer r.Unlock()
	prevAuth := g.authority
	g.authority = auth

	// Gate is in control, should it not be?
	if g == r.curr {
		t.From = g.state()
		t.From.Authority = prevAuth
		for existingGate := range r.gates {
			var (
				isGate     = existingGate == g
				higherAuth = existingGate.authority > g.authority
				betterPos  = existingGate.authority == g.authority && existingGate.position < g.position
			)
			if !isGate && (higherAuth || betterPos) {
				r.curr = existingGate
				t.From = g.state()
				t.To = existingGate.state()
				return t
			}
		}
		// No transfer happened, gate remains in control.
		t.To = g.state()
		return t
	}

	// Gate is not in control, should it be?
	higherAuth := g.authority > r.curr.authority
	betterPos := g.authority == r.curr.authority && g.position < r.curr.position
	if higherAuth || betterPos {
		t.From = r.curr.state()
		r.curr = g
		t.To = g.state()
	}
	return
}

// Config is the configuration for opening a controller.
type Config struct {
	alamos.Instrumentation
	// Concurrency specifies whether shared control is allowed over regions in the
	// controller.
	Concurrency control.Concurrency
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for opening a Controller.
	DefaultConfig = Config{Concurrency: control.Exclusive}
)

// Validate implements config.Config.
func (c Config) Validate() error { return nil }

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Concurrency = override.Numeric(c.Concurrency, other.Concurrency)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return other
}

// Controller controls access to a specified resource type over discrete time regions.
// A Controller maintains a set of regions that occupy non-overlapping time ranges.
// If the controller is open with control.Exclusive, each region can only have one
// subject, managed by a Gate, controlling it at a single time. Each region manages
// an independent set of gates that bid for control over the resource using a mix
// of both first-come-first-serve precedence and specified control authorities
// (control.Authority).
type Controller[R Resource] struct {
	Config
	mu      sync.RWMutex
	regions []*region[R]
}

// New creates a new Controller that controls access to the specified resource type.
func New[R Resource](cfg Config) (*Controller[R], error) {
	cfg, err := config.New(DefaultConfig, cfg)
	return &Controller[R]{Config: cfg, regions: []*region[R]{}}, err
}

// GateConfig is the configuration for opening a gate.
type GateConfig[R Resource] struct {
	// TimeRange sets the time range for the gate. Any subsequent calls to OpenGate
	// with overlapping time ranges will bind themselves to the same control region.
	// [REQUIRED]
	TimeRange telem.TimeRange
	// Authority sets the authority of the gate over the resource. For a complete
	// discussion of authority, see the package level documentation.
	// [REQUIRED]
	Authority control.Authority
	// Subject sets the identity of the gate, and is used to track changes in control
	// within the db.
	// [REQUIRED]
	Subject control.Subject
	// CreateResource is a callback that is called when the gate is opened. It should return
	// the resource that is being controlled. This is used to create the resource in the
	// database.
	// [REQUIRED}
	OpenResource func() (R, error)
	// ErrIfControlled indicates whether the controller should return an error if any
	// other gates are currently controlling the resource.
	// [OPTIONAL] Defaults to false.
	ErrIfControlled *bool
	// ErrOnUnauthorizedOpen indicates whether the controller should return an error
	// if the gate does not immediately take control when it is opened.
	// [OPTIONAL] Defaults to false.
	ErrOnUnauthorizedOpen *bool
}

var _ config.Config[GateConfig[Resource]] = GateConfig[Resource]{}

// DefaultGateConfig is the default configuration for opening a Gate.
func DefaultGateConfig[R Resource]() GateConfig[R] {
	return GateConfig[R]{
		ErrIfControlled:       config.False(),
		ErrOnUnauthorizedOpen: config.False(),
	}
}

// Validate implements config.Config.
func (c GateConfig[R]) Validate() error {
	v := validate.New("gate_config")
	validate.NotEmptyString(v, "subject.key", c.Subject.Key)
	validate.NonZeroable(v, "time_range", c.TimeRange)
	validate.NotNil(v, "open_resource", c.OpenResource)
	validate.NotNil(v, "err_if_controlled", c.ErrIfControlled)
	validate.NotNil(v, "err_on_unauthorized_open", c.ErrOnUnauthorizedOpen)
	return v.Error()
}

// Override implements config.Config.
func (c GateConfig[R]) Override(other GateConfig[R]) GateConfig[R] {
	c.Authority = override.Numeric(c.Authority, other.Authority)
	c.Subject.Key = override.String(c.Subject.Key, other.Subject.Key)
	c.Subject.Name = override.String(c.Subject.Name, other.Subject.Name)
	c.TimeRange.Start = override.Numeric(c.TimeRange.Start, other.TimeRange.Start)
	c.TimeRange.End = override.Numeric(c.TimeRange.End, other.TimeRange.End)
	c.OpenResource = override.Nil(c.OpenResource, other.OpenResource)
	c.ErrIfControlled = override.Nil(c.ErrIfControlled, other.ErrIfControlled)
	c.ErrOnUnauthorizedOpen = override.Nil(c.ErrOnUnauthorizedOpen, other.ErrOnUnauthorizedOpen)
	return c
}

// LeadingState returns the current control State of the leading region in the
// controller. Returns nil if no regions are under control.
func (c *Controller[R]) LeadingState() (state *State) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.regions) != 0 && len(c.regions[0].gates) != 0 {
		state = c.regions[0].curr.state()
	}
	return
}

// OpenGate opens a new gate for the region occupying the specified time range. If the
// region does not exist, it will be created and cfg.OpenResource will be called.
// If the region does exist, the new gate will be added to the authority chain for the
// existing region. If requiresUncontrolled is true, the region must not be controlled by
// another gate. If it is, an error will be returned.
func (c *Controller[R]) OpenGate(cfg GateConfig[R]) (g *Gate[R], t Transfer, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cfg, err = config.New(DefaultGateConfig[R](), cfg); err != nil {
		return
	}

	var exists bool
	for _, reg := range c.regions {
		// Check if there is an existing region that overlaps with that time range.
		if reg.timeRange.OverlapsWith(cfg.TimeRange) {
			// v1 optimization: one writer can only overlap with one region at any given time.
			if exists {
				err = errors.Newf("encountered multiple control regions for time range %s", cfg.TimeRange)
				c.L.DPanic(err.Error())
				return nil, t, err
			}
			// If there is an existing region, we open a new gate on that region.
			if g, t, err = reg.open(cfg); err != nil {
				return
			}
			exists = true
		}
	}
	if exists {
		return
	}
	var res R
	if res, err = cfg.OpenResource(); err != nil {
		return
	}
	reg := c.unsafeInsertNewRegion(cfg.TimeRange, res)
	g, t, err = reg.open(cfg)
	return
}

func (c *Controller[R]) unsafeInsertNewRegion(
	t telem.TimeRange,
	resource R,
) *region[R] {
	r := &region[R]{
		resource:   resource,
		gates:      make(map[*Gate[R]]struct{}),
		timeRange:  t,
		controller: c,
	}
	pos, _ := slices.BinarySearchFunc(c.regions, r, func(a *region[R], b *region[R]) int {
		return int(a.timeRange.Start - b.timeRange.Start)
	})
	c.regions = slices.Insert(c.regions, pos, r)
	return r
}

func (c *Controller[R]) remove(r *region[R]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, reg := range c.regions {
		if reg == r {
			c.regions = append(c.regions[:i], c.regions[i+1:]...)
			break
		}
	}
}
