// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

// Attachment is a kind of value that a caller attaches to a transaction. The Commit of
// the transaction delivers the values to the readers of DB.Observe. A transaction that
// closes without a commit delivers none. The values of two Attachments never mix, even
// when they have the same type.
type Attachment[T any] struct {
	// _ gives each Attachment its own address, which is its identity.
	_ byte
}

// NewAttachment returns a new Attachment. Its owner creates it once and holds it.
func NewAttachment[T any]() *Attachment[T] { return &Attachment[T]{} }

// Add attaches values to tx, after the values that a attached to tx before. Call it
// before tx commits, on the goroutine that writes through tx. It panics when tx is a
// DB.
func (a *Attachment[T]) Add(tx Tx, values ...T) {
	state := tx.txIdentity()
	if state == nil {
		panic("[gorp] - Attachment.Add needs a transaction, not a DB")
	}
	if len(values) == 0 {
		return
	}
	l := attachedBy(state.attached, a)
	if l == nil {
		l = &attachedList[T]{owner: a}
		state.attached = append(state.attached, l)
	}
	l.values = append(l.values, values...)
}

// Pending returns the values that a attached to tx so far, in the order that a attached
// them. Work registered with BeforeCommit can call it to read every value. Call it on
// the goroutine that writes through tx, and do not modify the values. It panics when tx
// is a DB.
func (a *Attachment[T]) Pending(tx Tx) []T {
	state := tx.txIdentity()
	if state == nil {
		panic("[gorp] - Attachment.Pending needs a transaction, not a DB")
	}
	if l := attachedBy(state.attached, a); l != nil {
		return l.values
	}
	return nil
}

// Of returns the values that a attached to the transaction of c, in the order that a
// attached them. The readers of c share the values, so a reader must not modify them.
func (a *Attachment[T]) Of(c Commit) []T {
	if l := attachedBy(c.attached, a); l != nil {
		return l.values
	}
	return nil
}

// attached is the values that one Attachment attached to a transaction. Each one is an
// *attachedList[T].
type attached interface{ attached() }

type attachedList[T any] struct {
	owner  *Attachment[T]
	values []T
}

func (*attachedList[T]) attached() {}

// attachedBy returns the values that a attached, or nil when it attached none.
func attachedBy[T any](lists []attached, a *Attachment[T]) *attachedList[T] {
	for _, l := range lists {
		if l, ok := l.(*attachedList[T]); ok && l.owner == a {
			return l
		}
	}
	return nil
}
