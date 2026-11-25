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
	"iter"
	"slices"
	"strings"

	"github.com/cespare/xxhash/v2"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// planned is a plan with what Apply needs to carry it out.
type planned struct {
	Plan
	// cancels maps the record of each restrict entry to the error of its cancel.
	cancels map[resource.ID]gorp.RestrictError
	// restricts are the records of the restrict entries, in the order that the delete
	// reaches them.
	restricts []resource.ID
	// batches hold the delete of each type of record, in the order that Apply deletes
	// them: a batch that removes a holder of a record of another batch comes first, and
	// otherwise the order that the types first appear in the delete.
	batches []batch
}

// batch is the delete of the records of one type.
type batch struct {
	deleter Deleter
	removed []resource.ID
	effects gorp.DeleteEffects
}

// compareEntries orders entries by record type, then key.
func compareEntries(a, b Entry) int {
	return cmp.Or(
		strings.Compare(string(a.Record.Type), string(b.Record.Type)),
		strings.Compare(a.Record.Key, b.Record.Key),
	)
}

// planDelete plans the delete of records, as seen by snap, through the Deleter of each
// type.
func (s *Service) planDelete(
	ctx context.Context,
	snap gorp.Snapshot,
	records []resource.ID,
) (planned, error) {
	var p planned
	for t, ids := range byType(records) {
		d, ok := s.cfg.Deleters[t]
		if !ok {
			return planned{}, errors.Wrapf(
				validate.ErrValidation,
				"impact cannot delete %s",
				ids[0],
			)
		}
		removed, effects, err := d.Plan(ctx, snap, ids)
		if err != nil {
			return planned{}, err
		}
		p.batches = append(
			p.batches,
			batch{deleter: d, removed: removed, effects: effects},
		)
	}
	// A Deleter plans its own type whole, so only another batch removes a holder.
	var removed map[resource.ID]int
	if len(p.batches) > 1 {
		removed = make(map[resource.ID]int)
		for i, b := range p.batches {
			for _, id := range b.removed {
				removed[id] = i
			}
		}
		if err := s.orderBatches(&p, removed); err != nil {
			return planned{}, err
		}
	}
	s.addEntries(&p, removed)
	return p, nil
}

// orderBatches orders p.batches so that each batch that removes a holder of a record of
// another batch comes before that batch. Batches with no such link keep their order.
// removed maps each record that p removes to the index of its batch. A cycle of links
// fails, since no order deletes each holder first.
func (s *Service) orderBatches(p *planned, removed map[resource.ID]int) error {
	n := len(p.batches)
	// holds[i*n+j] reports that batch i removes a holder of a record of batch j.
	holds := make([]bool, n*n)
	waits := make([]int, n)
	for j, b := range p.batches {
		for _, effects := range [][]gorp.DeleteEffect{
			b.effects.Restricts,
			b.effects.Cascades,
			b.effects.Detaches,
			b.effects.Weak,
		} {
			for _, e := range effects {
				i, ok := removed[s.cfg.RecordOf(e.Holder)]
				if ok && i != j && !holds[i*n+j] {
					holds[i*n+j] = true
					waits[j]++
				}
			}
		}
	}
	ordered := make([]batch, 0, n)
	placed := make([]bool, n)
	for range n {
		next := -1
		for i := range n {
			if !placed[i] && waits[i] == 0 {
				next = i
				break
			}
		}
		if next < 0 {
			return errors.New("[impact] - the records hold each other in a cycle")
		}
		placed[next] = true
		ordered = append(ordered, p.batches[next])
		for j := range n {
			if holds[next*n+j] {
				waits[j]--
			}
		}
	}
	p.batches = ordered
	return nil
}

// addEntries gives each record that holds a record of p.batches one entry, for the
// first of its effects in the order restrict, cascade, detach, and counts in p.Kept
// each other record that keeps a weak link. A holder in removed gets neither, since p
// deletes it.
func (s *Service) addEntries(p *planned, removed map[resource.ID]int) {
	var restricts, cascades, detaches int
	for _, b := range p.batches {
		restricts += len(b.effects.Restricts)
		cascades += len(b.effects.Cascades)
		detaches += len(b.effects.Detaches)
	}
	p.cancels = make(map[resource.ID]gorp.RestrictError, restricts)
	if n := restricts + cascades + detaches; n > 0 {
		p.Entries = make([]Entry, 0, n)
	}
	// effectOf holds the effect of each record with an entry, and no effect for each
	// record that keeps a weak link.
	effectOf := make(map[resource.ID]Effect, restricts+detaches)
	add := func(e gorp.DeleteEffect, effect Effect) (resource.ID, bool) {
		record := s.cfg.RecordOf(e.Holder)
		if effectOf[record] != "" || hasKey(removed, record) {
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
	for _, b := range p.batches {
		for _, e := range b.effects.Restricts {
			if id, ok := add(e, EffectRestrict); ok {
				p.restricts = append(p.restricts, id)
				p.cancels[id] = gorp.RestrictError{
					Target:    e.Target,
					Holder:    e.Holder,
					Reference: e.Reference,
				}
			}
		}
	}
	for _, b := range p.batches {
		for _, e := range b.effects.Cascades {
			add(e, EffectCascade)
		}
	}
	for _, b := range p.batches {
		for _, e := range b.effects.Detaches {
			add(e, EffectDetach)
		}
	}
	for _, b := range p.batches {
		for _, e := range b.effects.Weak {
			record := s.cfg.RecordOf(e.Holder)
			if !hasKey(effectOf, record) && !hasKey(removed, record) {
				effectOf[record] = ""
				p.Kept++
			}
		}
	}
	slices.SortFunc(p.Entries, compareEntries)
}

func hasKey[V any](m map[resource.ID]V, k resource.ID) bool {
	_, ok := m[k]
	return ok
}

// byType returns records grouped by type, the types in the order that they first
// appear.
func byType(records []resource.ID) iter.Seq2[resource.Type, []resource.ID] {
	return func(yield func(resource.Type, []resource.ID) bool) {
		var types []resource.Type
		of := make(map[resource.Type][]resource.ID, 1)
		for _, r := range records {
			if _, ok := of[r.Type]; !ok {
				types = append(types, r.Type)
			}
			of[r.Type] = append(of[r.Type], r)
		}
		for _, t := range types {
			if !yield(t, of[t]) {
				return
			}
		}
	}
}

// compareRecords orders records by type, then key.
func compareRecords(a, b resource.ID) int {
	return cmp.Or(
		strings.Compare(string(a.Type), string(b.Type)),
		strings.Compare(a.Key, b.Key),
	)
}

// tokenSum returns an xxHash-64 of p, the plan of the delete of records as the caller
// sees it: the records in sorted order, each visible entry's record and effect, and the
// hidden counts. Each field holds its length, so no two plans hash the same bytes. The
// token detects a stale view, not a forged one, so it needs no cryptographic hash.
func tokenSum(records []resource.ID, p Plan) uint64 {
	sorted := records
	if !slices.IsSortedFunc(records, compareRecords) {
		sorted = slices.SortedFunc(slices.Values(records), compareRecords)
	}
	sorted = slices.CompactFunc(slices.Clone(sorted), func(a, b resource.ID) bool {
		return compareRecords(a, b) == 0
	})
	var (
		fixed [256]byte
		buf   = fixed[:0]
	)
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(sorted)))
	for _, r := range sorted {
		buf = appendField(buf, string(r.Type))
		buf = appendField(buf, r.Key)
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

// token returns the hex form of tokenSum(records, p).
func token(records []resource.ID, p Plan) string {
	var (
		raw     [8]byte
		encoded [16]byte
	)
	binary.BigEndian.PutUint64(raw[:], tokenSum(records, p))
	hex.Encode(encoded[:], raw[:])
	return string(encoded[:])
}

// matches reports whether t is the token of p, the plan of the delete of records, with
// no hex form built.
func matches(t string, records []resource.ID, p Plan) bool {
	var decoded [8]byte
	if len(t) != hex.EncodedLen(len(decoded)) {
		return false
	}
	if _, err := hex.Decode(decoded[:], []byte(t)); err != nil {
		return false
	}
	return binary.BigEndian.Uint64(decoded[:]) == tokenSum(records, p)
}

// appendField appends the length of s, then s, to buf.
func appendField(buf []byte, s string) []byte {
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(s)))
	return append(buf, s...)
}
