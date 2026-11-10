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
	"iter"
	"slices"

	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/observe"
	"go.uber.org/zap"
)

// EntryChange is the net effect of a commit on one entry of a table.
type EntryChange[K Key, E Entry[K]] struct {
	// Key is the key of the entry.
	Key K
	// Value is the entry after the commit. It is the zero value for a delete.
	Value E
	// Kind is what the commit did to the entry.
	Kind ChangeKind
	// Derived is true when Gorp made every write to the entry by a cascade or a detach.
	Derived bool
}

// TxReader yields the changes that one commit made to the entries of a table.
type TxReader[K Key, E Entry[K]] = iter.Seq[EntryChange[K, E]]

// Observe returns an observable that notifies its caller of the changes that each
// commit of the DB makes to the entries of the table. The handlers run as the readers
// of DB.Observe do, so a handler must not commit to the DB on its own goroutine.
func (t *Table[K, E]) Observe() observe.Observable[TxReader[K, E]] {
	return observe.Translator[Commit, TxReader[K, E]]{
		Observable: t.db.Observe(),
		Translate: func(ctx context.Context, c Commit) (TxReader[K, E], bool) {
			if !slices.ContainsFunc(c.Changes, t.holds) {
				return nil, false
			}
			return t.Changes(ctx, c), true
		},
	}
}

// Changes returns the changes that c made to the entries of the table, in the order of
// c.Changes. A create or an update carries the key and the decoded entry. A delete
// carries the key only.
func (t *Table[K, E]) Changes(ctx context.Context, c Commit) TxReader[K, E] {
	return func(yield func(EntryChange[K, E]) bool) {
		for _, ch := range c.Changes {
			if !t.holds(ch) {
				continue
			}
			op, err := t.decode(ch)
			if err != nil {
				t.db.Instrumentation.L.DPanic(
					"failed to decode a change",
					zap.String("table", t.name),
					zap.String("key", ch.Record.Key),
					zap.Error(err),
				)
				continue
			}
			if !yield(op) {
				return
			}
		}
	}
}

// holds reports whether ch changed an entry of the table.
func (t *Table[K, E]) holds(ch Change) bool { return ch.Record.Table == t.name }

func (t *Table[K, E]) decode(ch Change) (EntryChange[K, E], error) {
	var (
		op  = EntryChange[K, E]{Kind: ch.Kind, Derived: ch.Derived}
		err error
	)
	if op.Key, err = key.Parse[K](ch.Record.Key); err != nil {
		return op, err
	}
	if ch.Kind == ChangeDelete {
		return op, nil
	}
	return op, t.db.Decode(ch.Value, &op.Value)
}
