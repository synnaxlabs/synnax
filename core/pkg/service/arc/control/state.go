// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package control folds the node control digest into the current control holder of each
// channel, so an Arc task can tell whether another writer out-ranks it.
package control

import (
	"sync"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// States is the current control holder of each channel, folded from the node control
// digest. It is safe for concurrent use.
type States struct {
	// mu guards states.
	mu sync.RWMutex
	// states maps a channel to the subject that currently holds it. A channel absent from
	// the map is uncontrolled.
	states map[channel.Key]xcontrol.State[channel.Key]
}

// New returns an empty control-state fold.
func New() *States {
	return &States{states: make(map[channel.Key]xcontrol.State[channel.Key])}
}

// Apply folds a control update into the state, recording acquired channels and removing
// released ones.
func (s *States) Apply(u xcontrol.Update[channel.Key]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range u.Transfers {
		switch {
		case t.IsAcquire(), t.IsTransfer():
			s.states[t.To.Resource] = *t.To
		case t.IsRelease():
			delete(s.states, t.From.Resource)
		}
	}
}

// ApplySeries folds every control update in a StringT digest series. A series of any other
// type is not a digest and is skipped.
func (s *States) ApplySeries(series telem.Series) error {
	if series.DataType != telem.StringT {
		return nil
	}
	updates, err := telem.UnmarshalJSONSeries[xcontrol.Update[channel.Key]](series)
	if err != nil {
		return errors.Wrap(err, "decode control digest series")
	}
	for _, u := range updates {
		s.Apply(u)
	}
	return nil
}

// Holder returns the current control state for a channel, if one exists.
func (s *States) Holder(key channel.Key) (xcontrol.State[channel.Key], bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	holder, ok := s.states[key]
	return holder, ok
}
