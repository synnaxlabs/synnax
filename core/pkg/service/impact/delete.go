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
	p.Token = token(keys, p.Entries)
	return p, nil
}

// token returns the hex form of the first 16 bytes of a SHA-256 hash of the delete of
// the channels with keys and its entries, each in sorted order.
func token(keys []channel.Key, entries []Entry) string {
	h := sha256.New()
	h.Write([]byte(DeleteChannelsChangeType))
	sorted := slices.Clone(keys)
	slices.SortFunc(sorted, func(a, b channel.Key) int { return cmp.Compare(a, b) })
	for _, k := range slices.Compact(sorted) {
		h.Write(binary.BigEndian.AppendUint32(nil, uint32(k)))
	}
	for _, e := range entries {
		h.Write([]byte{0})
		h.Write([]byte(e.ID))
		h.Write([]byte{0})
		h.Write([]byte(e.Effect))
	}
	return hex.EncodeToString(h.Sum(nil)[:16])
}
