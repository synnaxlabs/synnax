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
	// cancels maps the id of each restrict entry to the error that a cancel returns.
	cancels map[string]gorp.RestrictError
}

// planDelete plans the delete of the channels with keys, as seen by snap. Each record
// gets one entry, for the first of its effects in the order restrict, cascade, detach.
func (s *Service) planDelete(
	ctx context.Context,
	snap gorp.Snapshot,
	keys []channel.Key,
) (planned, error) {
	effects, err := s.cfg.Channel.PlanDelete(ctx, snap, keys)
	if err != nil {
		return planned{}, err
	}
	p := planned{
		Change:  Change{Variant: DeleteChannelsChange{Keys: keys}},
		cancels: make(map[string]gorp.RestrictError, len(effects.Restricts)),
	}
	byID := make(map[string]int)
	add := func(e gorp.DeleteEffect, effect Effect) (string, bool) {
		record := s.cfg.RecordOf(e.Holder)
		id := record.String()
		if _, ok := byID[id]; ok {
			return id, false
		}
		byID[id] = len(p.Entries)
		entry := Entry{ID: id, Record: record, Effect: effect}
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
		if id := s.cfg.RecordOf(e.Holder).String(); !slices.ContainsFunc(
			p.Entries,
			func(entry Entry) bool { return entry.ID == id },
		) {
			kept.Add(id)
		}
	}
	p.Kept = uint32(len(kept))
	slices.SortFunc(
		p.Entries,
		func(a, b Entry) int { return strings.Compare(a.ID, b.ID) },
	)
	return p, nil
}
