// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import (
	"context"
	"slices"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// ChangeKind is what a transaction did to a record. The zero value is invalid.
type ChangeKind uint8

const (
	// ChangeCreate sets a record that did not exist before the transaction.
	ChangeCreate ChangeKind = iota + 1
	// ChangeUpdate sets a record that existed before the transaction.
	ChangeUpdate
	// ChangeDelete deletes a record that existed before the transaction.
	ChangeDelete
)

// Change is the net effect of a transaction on one record.
type Change struct {
	// Record is the table and key of the record.
	Record Target
	// Kind is what the transaction did to the record.
	Kind ChangeKind
	// Derived is true when Gorp made every write to the record by a cascade or a
	// detach.
	Derived bool
	// Value is a copy of the encoded record at the last set. It is nil for a delete.
	Value []byte
	// Added holds the links that the transaction added to the record.
	Added []Link
	// Removed holds the links that the transaction removed from the record.
	Removed []Link
	// Links holds every link of the record after the transaction, or before it for a
	// delete.
	Links []Link
}

// Changes is what a transaction changed, one Change per record.
type Changes []Change

// ChangesOf returns what the typed writes through tx have changed so far, in the order
// tx first wrote each record. Cascades and detaches are included, and raw key-value
// writes are not: a kind compares against the record at the first typed write to it.
// It panics when tx is a DB.
func ChangesOf(tx Tx) Changes {
	state := tx.txIdentity()
	if state == nil {
		panic("[gorp] - ChangesOf needs a transaction, not a DB")
	}
	return state.changes.list()
}

// changeList merges the writes of one transaction into one entry per record.
type changeList struct {
	// at maps each record the transaction wrote to its position in entries.
	at      map[Target]int
	entries []changeEntry
}

type changeEntry struct {
	record Target
	// existed is true when the record existed at the first write to it.
	existed bool
	// present is true when the last write to the record was a set.
	present bool
	// derived is true when every write to the record was a cascade or a detach.
	derived bool
	value   []byte
	// before holds the links of the record at the first write to it.
	before []Link
	// after holds the links of the record at the last set.
	after []Link
}

// slot is the place of one write in a changeList. at is -1 on the first write to
// record.
type slot struct {
	at      int
	record  Target
	existed bool
	before  []Link
}

// find returns the slot of a write to record, which key stores in tx. On the first
// write to record, it reads the stored value of record. When record exists, find passes
// the stored value to held, which returns the links of record.
func (l *changeList) find(
	ctx context.Context,
	tx Tx,
	record Target,
	key []byte,
	held func(stored []byte) ([]Link, error),
) (slot, error) {
	if at, ok := l.at[record]; ok {
		return slot{at: at}, nil
	}
	s := slot{at: -1, record: record}
	stored, closer, err := tx.Get(ctx, key)
	if errors.Is(err, query.ErrNotFound) {
		return s, nil
	}
	if err != nil {
		return slot{}, err
	}
	s.existed = true
	if s.before, err = held(stored); err != nil {
		return slot{}, errors.Combine(err, closer.Close())
	}
	if err = closer.Close(); err != nil {
		return slot{}, err
	}
	return s, nil
}

// present reports whether the record of s exists at the write of s.
func (l *changeList) present(s slot) bool {
	if s.at >= 0 {
		return l.entries[s.at].present
	}
	return s.existed
}

// set merges a set of the record of s to value, which holds links. derived is true when
// the set is a cascade or a detach.
func (l *changeList) set(s slot, value []byte, links []Link, derived bool) {
	e := l.entry(s, derived)
	e.present = true
	e.value = slices.Clone(value)
	e.after = links
}

// delete merges a delete of the record of s. derived is true when the delete is a
// cascade.
func (l *changeList) delete(s slot, derived bool) {
	e := l.entry(s, derived)
	e.present = false
	e.value = nil
	e.after = nil
}

func (l *changeList) entry(s slot, derived bool) *changeEntry {
	if s.at < 0 {
		if l.at == nil {
			l.at = make(map[Target]int)
		}
		s.at = len(l.entries)
		l.at[s.record] = s.at
		l.entries = append(l.entries, changeEntry{
			record:  s.record,
			existed: s.existed,
			derived: true,
			before:  s.before,
		})
	}
	e := &l.entries[s.at]
	e.derived = e.derived && derived
	return e
}

func (l *changeList) list() Changes {
	var out Changes
	for _, e := range l.entries {
		kind := e.kind()
		if kind == 0 {
			continue
		}
		links := e.after
		if kind == ChangeDelete {
			links = e.before
		}
		out = append(out, Change{
			Record:  e.record,
			Kind:    kind,
			Derived: e.derived,
			Value:   e.value,
			Added:   subtract(e.after, e.before),
			Removed: subtract(e.before, e.after),
			Links:   subtract(links, nil),
		})
	}
	return out
}

// changed reports whether the transaction changed a record.
func (l *changeList) changed() bool {
	return slices.ContainsFunc(l.entries, func(e changeEntry) bool {
		return e.kind() != 0
	})
}

// kind returns what the transaction did to the record of e, or zero when the record is
// as it was before the transaction.
func (e changeEntry) kind() ChangeKind {
	switch {
	case e.present && e.existed:
		return ChangeUpdate
	case e.present:
		return ChangeCreate
	case e.existed:
		return ChangeDelete
	}
	return 0
}

// subtract returns the distinct links in links that are not in minus.
func subtract(links, minus []Link) []Link {
	var out []Link
	for _, l := range links {
		if !slices.Contains(minus, l) && !slices.Contains(out, l) {
			out = append(out, l)
		}
	}
	return out
}
