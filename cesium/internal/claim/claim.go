// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package claim tracks the time ranges that the writers of one channel hold. A writer
// that opens inside a live claim joins it and shares its resource. A claim holds no
// authority: every holder may use the resource.
package claim

import (
	"fmt"
	"slices"
	"sync"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// ErrClaimed is returned when an exclusive open reaches a claim.
var ErrClaimed = errors.New("time range claimed")

// Error is the refusal of an open or a resize that reaches another claim. A refused
// open wraps ErrClaimed, and a refused resize wraps validate.ErrValidation.
type Error struct {
	// TimeRange is the part of the other claim that the refused time range reaches.
	TimeRange telem.TimeRange
	// Open is true for a refused open.
	Open bool
}

// Error implements error.
func (e Error) Error() string {
	if e.Open {
		return fmt.Sprintf(
			"cannot access time range %v because a writer holds it: %v",
			e.TimeRange,
			ErrClaimed,
		)
	}
	return fmt.Sprintf(
		"cannot write to time range %v because another writer holds it: %v",
		e.TimeRange,
		validate.ErrValidation,
	)
}

// Unwrap returns the sentinel of the refusal.
func (e Error) Unwrap() error {
	if e.Open {
		return ErrClaimed
	}
	return validate.ErrValidation
}

// Config is the configuration of an open claim.
type Config[R any] struct {
	// Open opens the resource of a new region. Open runs only for the first claim of a
	// region, and with no lock held.
	Open func() (R, error)
	// TimeRange is the time range of the claim.
	TimeRange telem.TimeRange
	// Bounded makes the claim cover only its time range, which grows only through
	// Reach. A bounded claim is exclusive, and a claim that is not exclusive skips
	// bounded claims at open.
	Bounded bool
	// Exclusive refuses the open with ErrClaimed when the time range reaches a claim.
	Exclusive bool
}

// Set is the set of claims on one channel. The zero value is ready to use, and a Set is
// safe for concurrent use.
type Set[R any] struct {
	regions []*region[R]
	mu      sync.RWMutex
}

// Claim is the hold of one writer on a region of a Set.
type Claim[R any] struct{ region *region[R] }

type region[R any] struct {
	resource  R
	set       *Set[R]
	claims    set.Set[*Claim[R]]
	opening   chan struct{}
	timeRange telem.TimeRange
	bounded   bool
	// use serializes the holders of the region in Use.
	use sync.Mutex
}

// Open opens a claim on cfg.TimeRange. It joins the region the time range reaches, or
// opens a new region with cfg.Open when there is none.
func (s *Set[R]) Open(cfg Config[R]) (*Claim[R], error) {
	if cfg.Bounded {
		cfg.Exclusive = true
	}
	for {
		c, created, opening, err := s.joinOrCreate(cfg)
		if opening != nil {
			<-opening
			continue
		}
		if created != nil {
			return s.finishOpening(created, cfg)
		}
		return c, err
	}
}

func (s *Set[R]) joinOrCreate(cfg Config[R]) (
	c *Claim[R],
	created *region[R],
	opening <-chan struct{},
	err error,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var joined *region[R]
	for _, r := range s.regions {
		if (r.bounded && !cfg.Exclusive) || !r.timeRange.OverlapsWith(cfg.TimeRange) {
			continue
		}
		if r.opening != nil {
			return nil, nil, r.opening, nil
		}
		if cfg.Exclusive {
			return nil, nil, nil, Error{
				TimeRange: cfg.TimeRange.Intersection(r.timeRange),
				Open:      true,
			}
		}
		if joined != nil {
			return nil, nil, nil, errors.Newf(
				"time range %s reaches two claims",
				cfg.TimeRange,
			)
		}
		joined = r
	}
	if joined == nil {
		return nil, s.insertOpening(cfg.TimeRange, cfg.Bounded), nil, nil
	}
	return joined.add(cfg.TimeRange), nil, nil, nil
}

func (s *Set[R]) finishOpening(r *region[R], cfg Config[R]) (c *Claim[R], err error) {
	res, err := cfg.Open()
	s.mu.Lock()
	defer s.mu.Unlock()
	defer close(r.opening)
	r.opening = nil
	if err != nil {
		s.remove(r)
		return nil, err
	}
	r.resource = res
	return r.add(cfg.TimeRange), nil
}

func (s *Set[R]) insertOpening(tr telem.TimeRange, bounded bool) *region[R] {
	r := &region[R]{
		set:       s,
		claims:    set.New[*Claim[R]](),
		timeRange: tr,
		bounded:   bounded,
		opening:   make(chan struct{}),
	}
	pos, _ := slices.BinarySearchFunc(s.regions, r, func(a, b *region[R]) int {
		return int(a.timeRange.Start - b.timeRange.Start)
	})
	s.regions = slices.Insert(s.regions, pos, r)
	return r
}

func (s *Set[R]) remove(r *region[R]) {
	s.regions = slices.DeleteFunc(s.regions, func(o *region[R]) bool { return o == r })
}

// add adds a claim on tr to r. The caller must hold r.set.mu.
func (r *region[R]) add(tr telem.TimeRange) *Claim[R] {
	c := &Claim[R]{region: r}
	r.timeRange = r.timeRange.Union(tr)
	r.claims.Add(c)
	return c
}

// ResourceAt returns the resource of a region that tr reaches, preferring a bounded
// region, and false when there is none. It waits for a region that is opening.
func (s *Set[R]) ResourceAt(tr telem.TimeRange) (R, bool) {
	for {
		res, ok, opening := s.resourceAt(tr)
		if opening == nil {
			return res, ok
		}
		<-opening
	}
}

func (s *Set[R]) resourceAt(
	tr telem.TimeRange,
) (res R, ok bool, opening <-chan struct{}) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.regions {
		if !r.timeRange.OverlapsWith(tr) {
			continue
		}
		if r.opening != nil {
			opening = r.opening
			continue
		}
		res, ok = r.resource, true
		if r.bounded {
			return res, ok, nil
		}
	}
	return res, ok, opening
}

// Resource returns the resource of the claim's region. Only Use may change it.
func (c *Claim[R]) Resource() R {
	c.region.set.mu.RLock()
	defer c.region.set.mu.RUnlock()
	return c.region.resource
}

// Use calls f with the resource of the claim's region and returns its error. Calls to f
// are serialized across the claims of the region. f must not call Use on a claim of the
// same region.
func (c *Claim[R]) Use(f func(R) error) error {
	c.region.use.Lock()
	defer c.region.use.Unlock()
	return f(c.Resource())
}

// MoveStart moves the start of the claim's region to start, which frees any time before
// it. It fails when the region would reach another region. A bounded region only moves
// back, and a start at or after the end of the region does nothing.
func (c *Claim[R]) MoveStart(start telem.TimeStamp) error {
	s := c.region.set
	s.mu.Lock()
	defer s.mu.Unlock()
	r := c.region
	if start.Before(r.timeRange.Start) {
		return s.resize(r, start.Range(r.timeRange.End))
	}
	if !r.bounded && start.Before(r.timeRange.End) {
		r.timeRange.Start = start
	}
	return nil
}

// Reach moves the end of the claim's region forward to end. It does nothing when the
// region already reaches end, and fails when the region would reach another region.
func (c *Claim[R]) Reach(end telem.TimeStamp) error {
	s := c.region.set
	s.mu.Lock()
	defer s.mu.Unlock()
	if !end.After(c.region.timeRange.End) {
		return nil
	}
	return s.resize(c.region, c.region.timeRange.Start.Range(end))
}

// resize sets the time range of r to grown. The caller must hold s.mu.
func (s *Set[R]) resize(r *region[R], grown telem.TimeRange) error {
	for _, o := range s.regions {
		if o == r || (o.bounded && !r.bounded) {
			continue
		}
		if o.timeRange.OverlapsWith(grown) {
			return Error{TimeRange: grown.Intersection(o.timeRange)}
		}
	}
	r.timeRange = grown
	return nil
}

// Release releases the claim. It returns the resource of its region, and true when the
// claim was the last of the region, which then closes. A second Release returns false.
func (c *Claim[R]) Release() (R, bool) {
	r := c.region
	r.set.mu.Lock()
	defer r.set.mu.Unlock()
	if !r.claims.Contains(c) {
		var zero R
		return zero, false
	}
	r.claims.Remove(c)
	last := len(r.claims) == 0
	if last {
		r.set.remove(r)
	}
	return r.resource, last
}
