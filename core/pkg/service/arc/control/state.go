// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package control mirrors the C++ driver::control::States tracker. Full parity would lift
// it to a shared driver-level package and add the frame filter/all_authorized helpers.
package control

import (
	"sync"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
)

// States is the current control holder of each channel, fed by the node control digest.
// It is safe for concurrent use.
type States struct {
	mu     sync.RWMutex
	states map[channel.Key]xcontrol.State[channel.Key]
}

// New returns an empty control-state mirror.
func New() *States {
	return &States{states: make(map[channel.Key]xcontrol.State[channel.Key])}
}

// Apply folds a control update into the mirror, recording acquired channels and removing
// released ones.
func (s *States) Apply(u xcontrol.Update[channel.Key]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range u.Transfers {
		if t.To != nil {
			s.states[t.To.Resource] = *t.To
		} else if t.From != nil {
			delete(s.states, t.From.Resource)
		}
	}
}

// ApplySeries decodes and applies every control update in a StringT digest series.
func (s *States) ApplySeries(series telem.Series) {
	if series.DataType != telem.StringT {
		return
	}
	updates, err := telem.UnmarshalJSONSeries[xcontrol.Update[channel.Key]](series)
	if err != nil {
		return
	}
	for _, u := range updates {
		s.Apply(u)
	}
}

// ApplyIncrease optimistically records a strictly higher authority for a channel. Equal or
// lower authority is ignored, matching the Core's earlier-gate-wins tiebreak.
func (s *States) ApplyIncrease(
	subject xcontrol.Subject,
	key channel.Key,
	authority xcontrol.Authority,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.states[key]; ok && existing.Authority >= authority {
		return
	}
	s.states[key] = xcontrol.State[channel.Key]{
		Subject:   subject,
		Resource:  key,
		Authority: authority,
	}
}

// IsAuthorized reports whether subject holds the channel, or the channel is uncontrolled.
func (s *States) IsAuthorized(key channel.Key, subject xcontrol.Subject) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	holder, ok := s.states[key]
	return !ok || holder.Subject == subject
}

// Holder returns the current control state for a channel, if one exists.
func (s *States) Holder(key channel.Key) (xcontrol.State[channel.Key], bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	holder, ok := s.states[key]
	return holder, ok
}
