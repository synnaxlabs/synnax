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

// Resource represents some resource that can be controlled by a Gate. A Resource must have
// a ChannelKey that represents the resource that is being controlled.
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
	return c
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
	if err != nil {
		return nil, err
	}
	return &Controller[R]{Config: cfg, regions: []*region[R]{}}, nil
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
// existing region.
func (c *Controller[R]) OpenGate(cfg GateConfig[R]) (g *Gate[R], t Transfer, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cfg, err = config.New(DefaultGateConfig[R](), cfg); err != nil {
		return g, t, err
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
		return g, t, err
	}
	var res R
	if res, err = cfg.OpenResource(); err != nil {
		return
	}
	reg := c.unsafeInsertNewRegion(cfg.TimeRange, res)
	return reg.open(cfg)
}

func (c *Controller[R]) unsafeInsertNewRegion(
	t telem.TimeRange,
	resource R,
) *region[R] {
	r := &region[R]{
		resource:   resource,
		gates:      make(set.Set[*Gate[R]]),
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
			c.regions = slices.Delete(c.regions, i, i+1)
			break
		}
	}
}
