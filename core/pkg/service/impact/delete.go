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
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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
		if record := s.cfg.RecordOf(e.Holder); !hasKey(effectOf, record) {
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

// token returns the hex form of the first 16 bytes of a SHA-256 hash of the delete of
// the channels with keys, in sorted order, and its entries, which are sorted.
func token(keys []channel.Key, entries []Entry) string {
	h := sha256.New()
	h.Write([]byte(DeleteChannelsChangeType))
	sorted := slices.Clone(keys)
	slices.Sort(sorted)
	for _, k := range slices.Compact(sorted) {
		h.Write(binary.BigEndian.AppendUint32(nil, uint32(k)))
	}
	for _, e := range entries {
		h.Write([]byte{0})
		h.Write([]byte(e.Record.String()))
		h.Write([]byte{0})
		h.Write([]byte(e.Effect))
	}
	return hex.EncodeToString(h.Sum(nil)[:16])
}
