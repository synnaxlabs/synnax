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
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Resource is a resource that a Gate can control. Its key names it in control states
// and transfers.
type Resource[K comparable] interface {
	// Key returns the key of the resource.
	Key() K
}

// Config is the configuration for opening a controller.
type Config struct {
	alamos.Instrumentation
	// Concurrency specifies whether shared control is allowed over regions in the
	// controller.
	Concurrency Concurrency
}

var _ config.Config[Config] = Config{}

// Validate implements config.Config.
func (c Config) Validate() error { return nil }

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Concurrency = override.Numeric(c.Concurrency, other.Concurrency)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Controller controls access to a specified resource type over discrete time regions. A
// Controller maintains a set of regions that occupy non-overlapping time ranges. If the
// controller is open with ConcurrencyExclusive, each region can only have one subject,
// managed by a Gate, controlling it at a single time. Each region manages an
// independent set of gates that bid for control over the resource using a mix of both
// first-come-first-serve precedence and specified control authorities (Authority).
type Controller[K comparable, R Resource[K]] struct {
	Config
	regions []*region[K, R]
	mu      sync.RWMutex
}

// New creates a new Controller that controls access to the specified resource type.
func New[K comparable, R Resource[K]](cfg Config) (*Controller[K, R], error) {
	cfg, err := config.New(Config{
		Concurrency: ConcurrencyExclusive,
	}, cfg)
	if err != nil {
		return nil, err
	}
	return &Controller[K, R]{Config: cfg, regions: []*region[K, R]{}}, nil
}

// GateConfig is the configuration for opening a gate.
type GateConfig[K comparable, R Resource[K]] struct {
	// OpenResource opens the resource that the gate controls. The controller calls it
	// when the gate opens a new region.
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
	// Subject sets the identity of the gate, and names it in control states and
	// transfers.
	//
	// [REQUIRED]
	Subject Subject
	// TimeRange sets the time range for the gate. Any subsequent calls to OpenGate with
	// overlapping time ranges will bind themselves to the same control region.
	//
	// [REQUIRED]
	TimeRange telem.TimeRange
	// Authority sets the authority of the gate over the resource. Among the gates of a
	// region, the one with the highest authority is in control, and on a tie, the one
	// that opened first.
	//
	// [REQUIRED]
	Authority Authority
}

var _ config.Config[GateConfig[int, Resource[int]]] = GateConfig[int, Resource[int]]{}

func defaultGateConfig[K comparable, R Resource[K]]() GateConfig[K, R] {
	return GateConfig[K, R]{
		ErrIfControlled:       new(false),
		ErrOnUnauthorizedOpen: new(false),
	}
}

// Validate implements config.Config.
func (c GateConfig[K, R]) Validate() error {
	v := validate.New("gate_config")
	v.NotEmptyString("subject.key", c.Subject.Key)
	v.NonZeroable("time_range", c.TimeRange)
	v.NotNil("open_resource", c.OpenResource)
	v.NotNil("err_if_controlled", c.ErrIfControlled)
	v.NotNil("err_on_unauthorized_open", c.ErrOnUnauthorizedOpen)
	return v.Error()
}

// Override implements config.Config.
func (c GateConfig[K, R]) Override(other GateConfig[K, R]) GateConfig[K, R] {
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
	return c
}

// LeadingState returns the current control State of the leading region in the
// controller. Returns nil if no regions are under control.
func (c *Controller[K, R]) LeadingState() *State[K] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, reg := range c.regions {
		if s := reg.state(); s != nil {
			return s
		}
	}
	return nil
}

// HolderAt returns the subject that controls a region that tr reaches, and false when
// no gate controls such a region.
func (c *Controller[K, R]) HolderAt(tr telem.TimeRange) (Subject, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, reg := range c.regions {
		if !reg.timeRange.OverlapsWith(tr) {
			continue
		}
		reg.RLock()
		curr := reg.curr
		reg.RUnlock()
		if curr != nil {
			return curr.subject, true
		}
	}
	return Subject{}, false
}

// OpenGate opens a new gate for the region occupying the specified time range. If the
// region exists, the new gate joins its authority chain. If not, OpenGate creates the
// region and calls cfg.OpenResource, which can block, while it holds no locks. Gates
// for a region that is opening wait for it. If cfg.OpenResource fails, the region is
// removed and a waiting gate opens the resource itself.
func (c *Controller[K, R]) OpenGate(
	cfg GateConfig[K, R],
) (g *Gate[K, R], t Transfer[K], err error) {
	if cfg, err = config.New(defaultGateConfig[K, R](), cfg); err != nil {
		return g, t, err
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
func (c *Controller[K, R]) joinOrCreate(cfg GateConfig[K, R]) (
	g *Gate[K, R],
	t Transfer[K],
	created *region[K, R],
	opening <-chan struct{},
	err error,
) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var exists bool
	for _, reg := range c.regions {
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
	return nil, t, c.unsafeInsertOpeningRegion(cfg.TimeRange), nil, nil
}

// finishOpening opens the resource of a region from joinOrCreate and its first gate. If
// the resource fails to open or panics, it removes the region.
func (c *Controller[K, R]) finishOpening(
	r *region[K, R],
	cfg GateConfig[K, R],
) (g *Gate[K, R], t Transfer[K], err error) {
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

func (c *Controller[K, R]) unsafeInsertOpeningRegion(t telem.TimeRange) *region[K, R] {
	r := &region[K, R]{
		gates:      make(set.Set[*Gate[K, R]]),
		timeRange:  t,
		controller: c,
		opening:    make(chan struct{}),
	}
	pos, _ := slices.BinarySearchFunc(c.regions, r, func(a, b *region[K, R]) int {
		return int(a.timeRange.Start - b.timeRange.Start)
	})
	c.regions = slices.Insert(c.regions, pos, r)
	return r
}

func (c *Controller[K, R]) unsafeRemove(r *region[K, R]) {
	c.regions = slices.DeleteFunc(c.regions, func(reg *region[K, R]) bool {
		return reg == r
	})
}
