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
	// Value is a copy of the encoded record at the last set. It is nil for a delete.
	Value []byte
}

// Changes is what a transaction changed, one Change per record.
type Changes []Change

// ChangesOf returns what the typed writes through tx have changed so far, in the order
// tx first wrote each record. Cascades and detaches are included, and raw key-value
// writes are not. It panics when tx is a DB.
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
	value   []byte
}

// slot is the place of one write in a changeList. at is -1 on the first write to
// record.
type slot struct {
	at      int
	record  Target
	existed bool
}

// find returns the slot of a write to record, which key stores in tx. On the first
// write to record, it reads whether record exists.
func (l *changeList) find(
	ctx context.Context,
	tx Tx,
	record Target,
	key []byte,
) (slot, error) {
	if at, ok := l.at[record]; ok {
		return slot{at: at}, nil
	}
	existed, err := keyExists(ctx, tx, key)
	return slot{at: -1, record: record, existed: existed}, err
}

// set merges a set of the record of s to value.
func (l *changeList) set(s slot, value []byte) {
	e := l.entry(s)
	e.present = true
	e.value = slices.Clone(value)
}

// delete merges a delete of the record of s.
func (l *changeList) delete(s slot) {
	e := l.entry(s)
	e.present = false
	e.value = nil
}

func (l *changeList) entry(s slot) *changeEntry {
	if s.at < 0 {
		if l.at == nil {
			l.at = make(map[Target]int)
		}
		s.at = len(l.entries)
		l.at[s.record] = s.at
		l.entries = append(l.entries, changeEntry{record: s.record, existed: s.existed})
	}
	return &l.entries[s.at]
}

func (l *changeList) list() Changes {
	var out Changes
	for _, e := range l.entries {
		if kind := e.kind(); kind != 0 {
			out = append(out, Change{Record: e.record, Kind: kind, Value: e.value})
		}
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

// keyExists reports whether key holds a value in tx.
func keyExists(ctx context.Context, tx Tx, key []byte) (bool, error) {
	_, closer, err := tx.Get(ctx, key)
	if errors.Is(err, query.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err = closer.Close(); err != nil {
		return false, err
	}
	return true, nil
}
