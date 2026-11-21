// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// planned is a plan with what Apply needs to carry it out.
type planned struct {
	Plan
	// cancels maps the record of each restrict entry to the error of its cancel.
	cancels map[resource.ID]gorp.RestrictError
	// keys are the keys of the channels that the delete removes.
	keys channel.Keys
	// effects is what the delete does to the records that hold the channels.
	effects gorp.DeleteEffects
}

// compareEntries orders entries by record type, then key.
func compareEntries(a, b Entry) int {
	return cmp.Or(
		strings.Compare(string(a.Record.Type), string(b.Record.Type)),
		strings.Compare(a.Record.Key, b.Record.Key),
	)
}

// planDelete plans the delete of the channels with keys, as seen by snap. Each record
// gets one entry, for the first of its effects in the order restrict, cascade, detach.
func (s *Service) planDelete(
	ctx context.Context,
	snap gorp.Snapshot,
	keys []channel.Key,
) (planned, error) {
	deleted, effects, err := s.cfg.Channel.PlanDelete(ctx, snap, keys)
	if err != nil {
		return planned{}, err
	}
	p := planned{
		cancels: make(map[resource.ID]gorp.RestrictError, len(effects.Restricts)),
		keys:    deleted,
		effects: effects,
	}
	n := len(effects.Restricts) + len(effects.Cascades) + len(effects.Detaches)
	if n > 0 {
		p.Entries = make([]Entry, 0, n)
	}
	// effectOf holds the effect of each record with an entry, and no effect for each
	// record that keeps a weak link.
	effectOf := make(
		map[resource.ID]Effect,
		len(effects.Restricts)+len(effects.Detaches),
	)
	add := func(e gorp.DeleteEffect, effect Effect) (resource.ID, bool) {
		record := s.cfg.RecordOf(e.Holder)
		if effectOf[record] != "" {
			return record, false
		}
		effectOf[record] = effect
		entry := Entry{Record: record, Effect: effect}
		if effect == EffectRestrict {
			entry.Choices = []Choice{ChoiceCancel, ChoiceDelete}
			entry.Default = new(ChoiceCancel)
		}
		p.Entries = append(p.Entries, entry)
		return record, true
	}
	for _, e := range effects.Restricts {
		if id, ok := add(e, EffectRestrict); ok {
			p.cancels[id] = gorp.RestrictError{
				Target:    e.Target,
				Holder:    e.Holder,
				Reference: e.Reference,
			}
		}
	}
	for _, e := range effects.Cascades {
		add(e, EffectCascade)
	}
	for _, e := range effects.Detaches {
		add(e, EffectDetach)
	}
	for _, e := range effects.Weak {
		// A holder with no resource type, such as a tombstone, is no caller's record.
		record := s.cfg.RecordOf(e.Holder)
		if record.Type.IsValid() && !hasKey(effectOf, record) {
			effectOf[record] = ""
			p.Kept++
		}
	}
	slices.SortFunc(p.Entries, compareEntries)
	return p, nil
}

func hasKey(m map[resource.ID]Effect, k resource.ID) bool {
	_, ok := m[k]
	return ok
}
