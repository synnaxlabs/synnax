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
	"encoding/binary"
	"encoding/hex"
	"slices"
	"strings"

	"github.com/cespare/xxhash/v2"
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

// tokenSum returns an xxHash-64 of p, the plan of the delete of the channels with keys
// as the caller sees it: the keys in sorted order, each visible entry's record and
// effect, and the hidden counts. Each field holds its length, so no two plans hash the
// same bytes. The token detects a stale view, not a forged one, so it needs no
// cryptographic hash.
func tokenSum(keys []channel.Key, p Plan) uint64 {
	sorted := keys
	for i := 1; i < len(keys); i++ {
		if keys[i] <= keys[i-1] {
			sorted = slices.Compact(slices.Sorted(slices.Values(keys)))
			break
		}
	}
	var (
		fixed [256]byte
		buf   = fixed[:0]
	)
	buf = appendField(buf, string(DeleteChannelsChangeType))
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(sorted)))
	for _, k := range sorted {
		buf = binary.BigEndian.AppendUint32(buf, uint32(k))
	}
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(p.Entries)))
	for _, e := range p.Entries {
		buf = appendField(buf, string(e.Record.Type))
		buf = appendField(buf, e.Record.Key)
		buf = appendField(buf, string(e.Effect))
	}
	buf = binary.BigEndian.AppendUint32(buf, p.Hidden.Restrict)
	buf = binary.BigEndian.AppendUint32(buf, p.Hidden.Cascade)
	buf = binary.BigEndian.AppendUint32(buf, p.Hidden.Detach)
	return xxhash.Sum64(buf)
}

// token returns the hex form of tokenSum(keys, p).
func token(keys []channel.Key, p Plan) string {
	var (
		raw     [8]byte
		encoded [16]byte
	)
	binary.BigEndian.PutUint64(raw[:], tokenSum(keys, p))
	hex.Encode(encoded[:], raw[:])
	return string(encoded[:])
}

// matches reports whether t is the token of p, the plan of the delete of the channels
// with keys, with no hex form built.
func matches(t string, keys []channel.Key, p Plan) bool {
	var decoded [8]byte
	if len(t) != hex.EncodedLen(len(decoded)) {
		return false
	}
	if _, err := hex.Decode(decoded[:], []byte(t)); err != nil {
		return false
	}
	return binary.BigEndian.Uint64(decoded[:]) == tokenSum(keys, p)
}

// appendField appends the length of s, then s, to buf.
func appendField(buf []byte, s string) []byte {
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(s)))
	return append(buf, s...)
}
