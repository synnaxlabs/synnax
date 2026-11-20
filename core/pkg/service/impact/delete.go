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
	"github.com/synnaxlabs/x/set"
)

// planned is a plan with what Apply needs to carry it out.
type planned struct {
	Plan
	// cancels maps the text form of the record of each restrict entry to the error that
	// a cancel returns.
	cancels map[string]gorp.RestrictError
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
		cancels: make(map[string]gorp.RestrictError, len(effects.Restricts)),
		keys:    deleted,
		effects: effects,
	}
	listed := set.New[string]()
	add := func(e gorp.DeleteEffect, effect Effect) (string, bool) {
		record := s.cfg.RecordOf(e.Holder)
		id := record.String()
		if listed.Contains(id) {
			return id, false
		}
		listed.Add(id)
		entry := Entry{Record: record, Effect: effect}
		if effect == EffectRestrict {
			entry.Choices = []Choice{ChoiceCancel, ChoiceDelete}
			entry.Default = new(ChoiceCancel)
		}
		p.Entries = append(p.Entries, entry)
		return id, true
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
	kept := set.New[string]()
	for _, e := range effects.Weak {
		if id := s.cfg.RecordOf(e.Holder).String(); !listed.Contains(id) {
			kept.Add(id)
		}
	}
	p.Kept = uint32(len(kept))
	slices.SortFunc(p.Entries, compareEntries)
	return p, nil
}
