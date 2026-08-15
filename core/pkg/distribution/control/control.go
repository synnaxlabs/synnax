// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package control makes the control state of a Synnax cluster readable and observable
// from any node. Control is arbitrated by the storage engine on each channel's
// leaseholder, so the Service fans retrieves out to leaseholders and merges peer
// subscriptions into a single cluster-wide stream of transfers.
package control

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/control"
)

type (
	// Subject is an entity that can hold control authority over a channel.
	Subject = control.Subject
	// State is the control state of a single channel: the subject holding authority
	// over it, and the level of that authority.
	State = control.State[channel.Key]
	// Transfer is a transition of control over a single channel. A nil From is an
	// acquire; a nil To is a release.
	Transfer = control.Transfer[channel.Key]
	// Update is a batch of transfers that a Core applied atomically.
	Update = control.Update[channel.Key]
)

// stateFromStorage returns nil when s is nil, so the acquire and release ends of a
// transfer survive the conversion.
func stateFromStorage(s *ts.ControlState) *State {
	if s == nil {
		return nil
	}
	return &State{
		Subject:   s.Subject,
		Resource:  channel.Key(s.Resource),
		Authority: s.Authority,
	}
}

func updateFromStorage(u ts.ControlUpdate) Update {
	return Update{Transfers: lo.Map(
		u.Transfers,
		func(t ts.ControlTransfer, _ int) Transfer {
			return Transfer{From: stateFromStorage(t.From), To: stateFromStorage(t.To)}
		},
	)}
}

// statesFromStorage drops releases: updates returned by ts.DB.ControlStates carry only
// the To end of each transfer.
func statesFromStorage(u ts.ControlUpdate) []State {
	states := make([]State, 0, len(u.Transfers))
	for _, t := range u.Transfers {
		if s := stateFromStorage(t.To); s != nil {
			states = append(states, *s)
		}
	}
	return states
}

// diff returns the transfers that carry prev to next. Both maps are keyed by the
// channel each state controls. A channel present in next but absent or held by a
// different subject in prev yields an acquire or transfer; a channel absent from next
// yields a release.
func diff(prev, next map[channel.Key]State) []Transfer {
	transfers := make([]Transfer, 0, len(next))
	for key, to := range next {
		from, ok := prev[key]
		if ok && from == to {
			continue
		}
		t := Transfer{To: &to}
		if ok {
			t.From = &from
		}
		transfers = append(transfers, t)
	}
	for key, from := range prev {
		if _, ok := next[key]; !ok {
			transfers = append(transfers, Transfer{From: &from})
		}
	}
	return transfers
}

func statesByChannel(states []State) map[channel.Key]State {
	m := make(map[channel.Key]State, len(states))
	for _, s := range states {
		m[s.Resource] = s
	}
	return m
}
