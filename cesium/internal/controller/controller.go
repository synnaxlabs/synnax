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
	// State is the control State of a gate over a channel bound resource.
	State = control.State[core.ChannelKey]
	// Transfer is a transfer of control over a channel bound resource.
	Transfer = control.Transfer[core.ChannelKey]
)

// Resource represents some resource that can be controlled by a Gate. A Resource must have
// a ChannelKey that represents the resource that is being controlled.
type Resource interface {
	// ChannelKey returns the key of the channel that is being controlled.
	ChannelKey() core.ChannelKey
}

// Gate controls access to an resource for a given region of time.
type Gate[R Resource] struct {
	subject     control.Subject
	authority   control.Authority
	region      *region[R]
	position    int64
	concurrency control.Concurrency
}

// Subject returns information about the subject controlling this gate.
func (g *Gate[R]) Subject() control.Subject { return g.subject }

// Authority returns the control authority for this gate.
func (g *Gate[R]) Authority() control.Authority { return g.authority }

// State returns the current control State of the gate.
func (g *Gate[R]) State() *State {
	return &State{
		Subject:   g.subject,
		Resource:  g.region.resource.ChannelKey(),
		Authority: g.authority,
	}
}

// PeekResource returns the resource that is controlled by the gate. The resource is NOT valid
// for modification or use.
func (g *Gate[R]) PeekResource() R {
	g.region.RLock()
	defer g.region.RUnlock()
	return g.region.resource
}

// Authorized authorizes the gates access to the resource. If another gate has precedence,
// Authorized will return false.
func (g *Gate[R]) Authorized() (e R, ok bool) {
	e, err := g.Authorize()
	return e, err == nil
}

// Authorize authorizes the gate's access to the resource. If another gate has precedence,
// Authorize will return a control.Unauthorized error, and the zero value for the resource.
// If the gate is the current gate, returns the resource and a nil error.
func (g *Gate[R]) Authorize() (r R, err error) {
	g.region.RLock()
	defer g.region.RUnlock()
	// In the case of exclusive concurrency, we only need to check if the gate is the
	// current gate.
	var ok bool
	if g.concurrency == control.Exclusive {
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
// region (i.e. transfer.IsRelease() == true), the resource will be returned. Otherwise,
// the zero value of the resource will be returned.
func (g *Gate[R]) Release() (resource R, transfer Transfer) { return g.region.release(g) }

// SetAuthority changes the gate's authority, returning any transfer of control that
// may have occurred as a result.
func (g *Gate[R]) SetAuthority(auth control.Authority) Transfer {
	return g.region.update(g, auth)
}

type region[R Resource] struct {
	sync.RWMutex
	timeRange  telem.TimeRange
	resource   R
	counter    int64
	curr       *Gate[R]
	gates      map[*Gate[R]]struct{}
	controller *Controller[R]
}

// open opens a new gate on the region with the given config.
func (r *region[R]) open(
	cfg GateConfig[R],
	con control.Concurrency,
) (g *Gate[R], t Transfer, err error) {
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
		region:      r,
		subject:     cfg.Subject,
		authority:   cfg.Authority,
		position:    r.counter,
		concurrency: con,
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
			t.From = r.curr.State()
		}
		r.curr = g
		t.To = g.State()
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
func (r *region[R]) release(g *Gate[R]) (e R, transfer Transfer) {
	r.Lock()
	r.Unlock()
	e, transfer = r.unprotectedRelease(g)
	if transfer.IsRelease() {
		r.controller.remove(r)
	}
	return
}

// update a gate's authority.
func (r *region[R]) update(g *Gate[R], auth control.Authority) Transfer {
	r.Lock()
	t := r.unprotectedUpdate(g, auth)
	r.Unlock()
	return t
}

func (r *region[R]) unprotectedUpdate(
	g *Gate[R],
	auth control.Authority,
) (t Transfer) {
	prevAuth := g.authority
	g.authority = auth

	// Gate is in control, should it not be?
	if g == r.curr {
		t.From = g.State()
		t.From.Authority = prevAuth
		for existingGate := range r.gates {
			var (
				isGate     = existingGate == g
				higherAuth = existingGate.authority > g.authority
				betterPos  = existingGate.authority == g.authority && existingGate.position < g.position
			)
			if !isGate && (higherAuth || betterPos) {
				r.curr = existingGate
				t.From = g.State()
				t.To = existingGate.State()
				return t
			}
		}
		// No transfer happened, gate remains in control.
		t.To = g.State()
		return t
	}

	// Gate is not in control, should it be?
	higherAuth := g.authority > r.curr.authority
	betterPos := g.authority == r.curr.authority && g.position < r.curr.position
	if higherAuth || betterPos {
		t.From = r.curr.State()
		r.curr = g
		t.To = g.State()
	}
	return
}

// unprotectedRelease releases a gate from the region without locking.
// If the gate is the last gate in the region, the region will be removed from
// the controller.
func (r *region[R]) unprotectedRelease(g *Gate[R]) (res R, t Transfer) {
	delete(r.gates, g)
	if wasInControl := r.curr == g; !wasInControl {
		return
	}
	r.curr = nil
	t.From = g.State()
	for existingG := range r.gates {
		if r.curr == nil {
			r.curr = existingG
			t.To = existingG.State()
		} else {
			// Three cases here: no one is in control, provided gate has higher authority,
			// provided gate has equal authority and a higher position.
			higherAuth := existingG.authority > r.curr.authority
			betterPos := existingG.authority == r.curr.authority && existingG.position < r.curr.position
			if higherAuth || betterPos {
				r.curr = existingG
				t.To = existingG.State()
			}
		}
	}
	return r.resource, t
}

// Config is the configuration for opening a controller.
type Config struct {
	alamos.Instrumentation
	Concurrency control.Concurrency
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for opening a Controller.
	DefaultConfig = Config{Concurrency: control.Exclusive}
)

func (c Config) Validate() error {
	return nil
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Concurrency = override.Numeric(c.Concurrency, other.Concurrency)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return other
}

type Controller[R Resource] struct {
	Config
	mu      sync.RWMutex
	regions []*region[R]
}

func New[R Resource](cfg Config) (*Controller[R], error) {
	cfg, err := config.New(DefaultConfig, cfg)
	return &Controller[R]{
		Config:  cfg,
		regions: []*region[R]{},
	}, err
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

var (
	_ config.Config[GateConfig[Resource]] = GateConfig[Resource]{}
)

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
		state = c.regions[0].curr.State()
	}
	return
}

// OpenGate opens a new gate for the region occupying the specified time range. If the
// region does not exist, it will be created and cfg.OpenResource will be called.
// If the region does exist, the new gate will be added to the authority chain for the
// existing region. If requiresUncontrolled is true, the region must not be controlled by
// another gate. If it is, an error will be returned.
func (c *Controller[R]) OpenGate(
	cfg GateConfig[R],
) (g *Gate[R], t Transfer, err error) {
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
			if g, t, err = reg.open(cfg, c.Concurrency); err != nil {
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
	reg := c.insertNewRegion(cfg.TimeRange, res)
	if g, t, err = reg.open(cfg, c.Concurrency); err != nil {
		return
	}
	return
}

func (c *Controller[R]) insertNewRegion(
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
	for i, reg := range c.regions {
		if reg == r {
			c.regions = append(c.regions[:i], c.regions[i+1:]...)
			break
		}
	}
	c.mu.Unlock()
}
