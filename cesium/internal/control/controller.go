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
	"slices"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type (
	// State is the control State of a gate over a channel-bound resource.
	State = control.State[channel.Key]
	// Transfer is a transfer of control over a channel-bound resource.
	Transfer = control.Transfer[channel.Key]
)

// Resource represents some resource that can be controlled by a Gate. A Resource must
// have a ChannelKey that represents the resource that is being controlled.
type Resource interface {
	// ChannelKey returns the key of the channel that is being controlled.
	ChannelKey() channel.Key
}

// Config is the configuration for opening a controller.
type Config struct {
	alamos.Instrumentation
	// Concurrency specifies whether shared control is allowed over regions in the
	// controller.
	Concurrency control.Concurrency
}

var (
	_             config.Config[Config] = Config{}
	defaultConfig                       = Config{
		Concurrency: control.ConcurrencyExclusive,
	}
)

// Validate implements config.Config.
func (c Config) Validate() error { return nil }

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Concurrency = override.Numeric(c.Concurrency, other.Concurrency)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Controller controls access to a specified resource type over discrete time regions. A
// Controller maintains a set of regions that occupy non-overlapping time ranges, except
// that an unbounded region can overlap a bounded one. If the controller is open with
// control.ConcurrencyExclusive, each region can only have one subject, managed by a
// Gate, controlling it at a single time. Each region manages an independent set of
// gates that bid for control over the resource using a mix of both
// first-come-first-serve precedence and specified control authorities
// (control.Authority).
type Controller[R Resource] struct {
	Config
	regions []*region[R]
	mu      sync.RWMutex
}

// New creates a new Controller that controls access to the specified resource type.
func New[R Resource](cfg Config) (*Controller[R], error) {
	cfg, err := config.New(defaultConfig, cfg)
	if err != nil {
		return nil, err
	}
	return &Controller[R]{Config: cfg, regions: []*region[R]{}}, nil
}

// GateConfig is the configuration for opening a gate.
type GateConfig[R Resource] struct {
	// CreateResource is a callback that is called when the gate is opened. It should
	// return the resource that is being controlled. This is used to create the resource
	// in the database.
	//
	// [REQUIRED]
	OpenResource func() (R, error)
	// ErrIfControlled indicates whether the controller should return an error if any
	// other gates are currently controlling the resource.
	//
	// [OPTIONAL] Defaults to false.
	ErrIfControlled *bool
	// ErrOnUnauthorizedOpen indicates whether the controller should return an error if
	// the gate does not immediately take control when it is opened.
	//
	// [OPTIONAL] Defaults to false.
	ErrOnUnauthorizedOpen *bool
	// Bounded marks a gate that holds only TimeRange, and grows through Gate.Reach. A
	// bounded gate is alone in its region: it fails to open with
	// control.ErrUnauthorized where another region holds TimeRange, and no gate joins
	// it. An unbounded region can open or extend over it, and its next Gate.Reach into
	// that region fails. A bounded region does not count toward LeadingState.
	//
	// [OPTIONAL] Defaults to false.
	Bounded *bool
	// Subject sets the identity of the gate, and is used to track changes in control
	// within the db.
	//
	// [REQUIRED]
	Subject control.Subject
	// TimeRange sets the time range for the gate. Any subsequent calls to OpenGate with
	// overlapping time ranges will bind themselves to the same control region.
	//
	// [REQUIRED]
	TimeRange telem.TimeRange
	// Authority sets the authority of the gate over the resource. For a complete
	// discussion of authority, see the package level documentation.
	//
	// [REQUIRED]
	Authority control.Authority
}

var _ config.Config[GateConfig[Resource]] = GateConfig[Resource]{}

func defaultGateConfig[R Resource]() GateConfig[R] {
	return GateConfig[R]{
		ErrIfControlled:       new(false),
		ErrOnUnauthorizedOpen: new(false),
		Bounded:               new(false),
	}
}

// Validate implements config.Config.
func (c GateConfig[R]) Validate() error {
	v := validate.New("gate_config")
	v.NotEmptyString("subject.key", c.Subject.Key)
	v.NonZeroable("time_range", c.TimeRange)
	v.NotNil("open_resource", c.OpenResource)
	v.NotNil("err_if_controlled", c.ErrIfControlled)
	v.NotNil("err_on_unauthorized_open", c.ErrOnUnauthorizedOpen)
	v.NotNil("bounded", c.Bounded)
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
	c.ErrOnUnauthorizedOpen = override.Nil(
		c.ErrOnUnauthorizedOpen,
		other.ErrOnUnauthorizedOpen,
	)
	c.Bounded = override.Nil(c.Bounded, other.Bounded)
	return c
}

// LeadingState returns the current control State of the leading region in the
// controller, skipping bounded regions. Returns nil if no regions are under control.
func (c *Controller[R]) LeadingState() *State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, reg := range c.regions {
		if s := reg.state(); s != nil {
			return s
		}
	}
	return nil
}

// ResourceAt returns the resource held by the region overlapping tr, reporting false
// when no region covers it. A bounded region takes precedence over an unbounded region
// that overlaps it. ResourceAt waits for a region that is opening unless an open
// bounded region covers tr, so a caller must not hold a lock that the resource's opener
// needs. The returned resource is safe to read only through fields that are themselves
// safe for concurrent access, as the controller's lock is released before the caller
// touches it.
func (c *Controller[R]) ResourceAt(tr telem.TimeRange) (R, bool) {
	for {
		res, ok, opening := c.resourceAt(tr)
		if opening == nil {
			return res, ok
		}
		<-opening
	}
}

func (c *Controller[R]) resourceAt(
	tr telem.TimeRange,
) (res R, ok bool, opening <-chan struct{}) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, reg := range c.regions {
		if !reg.timeRange.OverlapsWith(tr) {
			continue
		}
		if reg.opening != nil {
			opening = reg.opening
			continue
		}
		res, ok = reg.resource, true
		if reg.bounded {
			return res, ok, nil
		}
	}
	return res, ok, opening
}

// OpenGate opens a new gate for the region occupying the specified time range. If the
// region exists, the new gate joins its authority chain. If not, OpenGate creates the
// region and calls cfg.OpenResource, which can block, while it holds no locks. Gates
// for a region that is opening wait for it. If cfg.OpenResource fails, the region is
// removed and a waiting gate opens the resource itself.
func (c *Controller[R]) OpenGate(
	cfg GateConfig[R],
) (g *Gate[R], t Transfer, err error) {
	if cfg, err = config.New(defaultGateConfig[R](), cfg); err != nil {
		return g, t, err
	}
	if *cfg.Bounded {
		cfg.ErrIfControlled = new(true)
	}
	for {
		g, t, created, opening, err := c.joinOrCreate(cfg)
		if opening != nil {
			<-opening
			continue
		}
		if created != nil {
			return c.finishOpening(created, cfg)
		}
		return g, t, err
	}
}

// joinOrCreate opens a gate on the region overlapping cfg.TimeRange. If that region is
// still opening, it returns the channel that closes when the region finishes. If no
// region overlaps, it returns a new region that the caller must pass to finishOpening.
func (c *Controller[R]) joinOrCreate(cfg GateConfig[R]) (
	g *Gate[R],
	t Transfer,
	created *region[R],
	opening <-chan struct{},
	err error,
) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var exists bool
	for _, reg := range c.regions {
		if reg.bounded && !*cfg.ErrIfControlled {
			continue
		}
		// Check if there is an existing region that overlaps with that time range.
		if reg.timeRange.OverlapsWith(cfg.TimeRange) {
			// v1 optimization: one writer can only overlap with one region at any given
			// time.
			if exists {
				err = errors.Newf(
					"encountered multiple control regions for time range %s",
					cfg.TimeRange,
				)
				c.L.DPanic(err.Error())
				return nil, t, nil, nil, err
			}
			if reg.opening != nil {
				return nil, t, nil, reg.opening, nil
			}
			// If there is an existing region, we open a new gate on that region.
			if g, t, err = reg.open(cfg); err != nil {
				return g, t, nil, nil, err
			}
			exists = true
		}
	}
	if exists {
		return g, t, nil, nil, nil
	}
	return nil, t, c.unsafeInsertOpeningRegion(cfg.TimeRange, *cfg.Bounded), nil, nil
}

// finishOpening opens the resource of a region from joinOrCreate and its first gate. If
// the resource fails to open or panics, it removes the region.
func (c *Controller[R]) finishOpening(
	r *region[R],
	cfg GateConfig[R],
) (g *Gate[R], t Transfer, err error) {
	var (
		res    R
		opened bool
	)
	defer func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		opening := r.opening
		r.opening = nil
		defer close(opening)
		if !opened {
			c.unsafeRemove(r)
			return
		}
		r.resource = res
		g, t, err = r.open(cfg)
	}()
	res, err = cfg.OpenResource()
	opened = err == nil
	return g, t, err
}

func (c *Controller[R]) moveStart(r *region[R], start telem.TimeStamp) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if start.Before(r.timeRange.Start) {
		return c.unsafeResize(r, start.Range(r.timeRange.End))
	}
	r.Lock()
	defer r.Unlock()
	if !r.bounded && start.Before(r.timeRange.End) {
		r.timeRange.Start = start
	}
	return nil
}

func (c *Controller[R]) reach(r *region[R], end telem.TimeStamp) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !end.After(r.timeRange.End) {
		return nil
	}
	return c.unsafeResize(r, r.timeRange.Start.Range(end))
}

func (c *Controller[R]) unsafeResize(r *region[R], grown telem.TimeRange) error {
	for _, reg := range c.regions {
		if reg == r || (reg.bounded && !r.bounded) {
			continue
		}
		if reg.timeRange.OverlapsWith(grown) {
			return reg.overlapError(grown)
		}
	}
	r.Lock()
	defer r.Unlock()
	r.timeRange = grown
	return nil
}

func (c *Controller[R]) unsafeInsertOpeningRegion(
	t telem.TimeRange,
	bounded bool,
) *region[R] {
	r := &region[R]{
		gates:      make(set.Set[*Gate[R]]),
		timeRange:  t,
		controller: c,
		bounded:    bounded,
		opening:    make(chan struct{}),
	}
	pos, _ := slices.BinarySearchFunc(c.regions, r, func(a, b *region[R]) int {
		return int(a.timeRange.Start - b.timeRange.Start)
	})
	c.regions = slices.Insert(c.regions, pos, r)
	return r
}

func (c *Controller[R]) unsafeRemove(r *region[R]) {
	c.regions = slices.DeleteFunc(c.regions, func(reg *region[R]) bool {
		return reg == r
	})
}
