// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package crdt implements a replicated text sequence: a conflict-free data type that
// lets many replicas concurrently edit a single string and always converge to the same
// value. The merge rule is the non-interleaving list CRDT in which each character is a
// node in a tree, anchored to an existing character on its left or right, so that
// concurrent runs of text inserted at the same position are never interleaved.
//
// The Insert, Delete, and ID types are generated from schemas/crdt.oracle and shared
// with the TypeScript implementation (@synnaxlabs/x crdt); the two runtimes materialize
// identically, locked by the conformance vectors in testdata.
package crdt

import (
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/spatial"
)

// isRoot reports whether id is the document root sentinel. The zero replica is reserved
// for the root and never assigned to a real replica.
func isRoot(id ID) bool { return id.Replica == 0 && id.Counter == 0 }

// idLess reports whether a sorts before b, ordering by replica then counter. The order
// is the tiebreak between characters that share an anchor, so every replica resolves
// concurrent insertions identically.
func idLess(a, b ID) bool {
	if a.Replica != b.Replica {
		return a.Replica < b.Replica
	}
	return a.Counter < b.Counter
}

// element is a node in the document tree. The in-order traversal of the tree, skipping
// deleted nodes, yields the materialized document.
type element struct {
	id      ID
	char    int32
	deleted bool
	// left and right hold the children anchored on each side, each kept sorted by id.
	left  []*element
	right []*element
}

// Text is a replicated text document. A Text is owned by a single replica, identified
// at construction; local edits are attributed to that replica. Text is not safe for
// concurrent use.
type Text struct {
	replica  uint32
	counter  uint64
	root     *element
	elements map[ID]*element
	// tombstones holds ids deleted before their insert was seen, so the delete can be
	// applied to the character once it arrives.
	tombstones set.Set[ID]
	// pending holds inserts whose origin has not yet been integrated, buffered until the
	// origin arrives.
	pending []Insert
	// order caches the in-order traversal of every element, rebuilt lazily when dirty.
	order []*element
	dirty bool
}

// New creates an empty document owned by the given replica. The replica must be non-zero
// and unique among the replicas editing the document.
func New(replica uint32) *Text {
	return &Text{
		replica:    replica,
		root:       &element{},
		elements:   make(map[ID]*element),
		tombstones: set.New[ID](),
	}
}

// Replica returns the id of the replica that owns this document.
func (t *Text) Replica() uint32 { return t.replica }

// Snapshot captures the full current state of the document as the operations that
// reconstruct it when applied to an empty document, in an order where every operation's
// origin precedes it. It is used to bootstrap a replica joining an in-progress session.
// Deleted characters are included as both an insert and a delete so anchoring is
// preserved.
func (t *Text) Snapshot() (inserts []Insert, deletes []Delete) {
	var walk func(e *element, origin ID)
	walk = func(e *element, origin ID) {
		if e != t.root {
			inserts = append(inserts, Insert{ID: e.id, Origin: origin, Side: e.side, Char: e.char})
			if e.deleted {
				deletes = append(deletes, Delete{ID: e.id})
			}
		}
		for _, c := range e.left {
			walk(c, e.id)
		}
		for _, c := range e.right {
			walk(c, e.id)
		}
	}
	walk(t.root, ID{})
	return inserts, deletes
}

// Load applies a snapshot to the document. It is intended for a freshly created
// document; the local replica's edits remain attributed to its own replica id and do not
// collide with the snapshot's operations.
func (t *Text) Load(inserts []Insert, deletes []Delete) {
	t.ApplyInsert(inserts...)
	t.ApplyDelete(deletes...)
}

// rebuild regenerates the cached in-order traversal if it is stale.
func (t *Text) rebuild() {
	if !t.dirty && t.order != nil {
		return
	}
	t.order = t.order[:0]
	t.walk(t.root)
	t.dirty = false
}

// walk appends the in-order traversal of the subtree rooted at e to t.order, excluding
// the root sentinel itself.
func (t *Text) walk(e *element) {
	for _, c := range e.left {
		t.walk(c)
	}
	if e != t.root {
		t.order = append(t.order, e)
	}
	for _, c := range e.right {
		t.walk(c)
	}
}

// visible returns the live characters in document order.
func (t *Text) visible() []*element {
	t.rebuild()
	out := make([]*element, 0, len(t.order))
	for _, e := range t.order {
		if !e.deleted {
			out = append(out, e)
		}
	}
	return out
}

// Len returns the number of live characters in the document.
func (t *Text) Len() int { return len(t.visible()) }

// String materializes the document into its current string value.
func (t *Text) String() string {
	vis := t.visible()
	runes := make([]rune, len(vis))
	for i, e := range vis {
		runes[i] = rune(e.char)
	}
	return string(runes)
}

// IndexToID returns the id of the character at the given live index. The second return
// is false if the index is out of range.
func (t *Text) IndexToID(index int) (ID, bool) {
	vis := t.visible()
	if index < 0 || index >= len(vis) {
		return ID{}, false
	}
	return vis[index].id, true
}

// Insert inserts text at the given live index and returns the operations that describe
// the edit. The operations are applied to this document before they are returned, and
// must be broadcast to other replicas for them to converge. index is measured in code
// points and may equal Len to append.
func (t *Text) Insert(index int, text string) []Insert {
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}
	vis := t.visible()
	var left *element
	if index <= 0 {
		left = t.root
	} else if index-1 < len(vis) {
		left = vis[index-1]
	} else if len(vis) > 0 {
		left = vis[len(vis)-1]
	} else {
		left = t.root
	}
	var right *element
	if index >= 0 && index < len(vis) {
		right = vis[index]
	}
	ops := make([]Insert, 0, len(runes))
	for _, r := range runes {
		var origin ID
		var side spatial.XLocation
		if right != nil && len(left.right) > 0 {
			origin, side = right.id, spatial.XLocationLeft
		} else {
			origin, side = left.id, spatial.XLocationRight
		}
		t.counter++
		op := Insert{
			ID:     ID{Replica: t.replica, Counter: t.counter},
			Origin: origin,
			Side:   side,
			Char:   r,
		}
		left = t.place(op)
		ops = append(ops, op)
	}
	return ops
}

// Delete removes length characters starting at the given live index and returns the
// operations that describe the edit. The operations are applied to this document before
// they are returned.
func (t *Text) Delete(index, length int) []Delete {
	if length <= 0 {
		return nil
	}
	vis := t.visible()
	ops := make([]Delete, 0, length)
	for i := index; i < index+length && i < len(vis); i++ {
		op := Delete{ID: vis[i].id}
		t.ApplyDelete(op)
		ops = append(ops, op)
	}
	return ops
}

// ApplyInsert integrates insert operations produced by other replicas. Operations may
// be supplied in any order and duplicates are ignored; operations whose origin has not
// yet arrived are buffered and integrated once it does.
func (t *Text) ApplyInsert(ops ...Insert) {
	for _, op := range ops {
		if t.place(op) != nil {
			t.drain()
		} else {
			t.pending = append(t.pending, op)
		}
	}
}

// ApplyDelete integrates delete operations produced by other replicas. A delete whose
// character has not yet been seen is recorded so the character is tombstoned on arrival.
func (t *Text) ApplyDelete(ops ...Delete) {
	for _, op := range ops {
		if e, ok := t.elements[op.ID]; ok {
			if !e.deleted {
				e.deleted = true
				t.dirty = true
			}
			continue
		}
		t.tombstones.Add(op.ID)
	}
}

// place attaches an insert to the tree, returning the created element. It returns nil
// without buffering when the operation was already integrated or its origin is not yet
// present.
func (t *Text) place(op Insert) *element {
	if e, ok := t.elements[op.ID]; ok {
		return e
	}
	origin := t.root
	if !isRoot(op.Origin) {
		var ok bool
		if origin, ok = t.elements[op.Origin]; !ok {
			return nil
		}
	}
	e := &element{id: op.ID, char: op.Char}
	if t.tombstones.Contains(op.ID) {
		e.deleted = true
		t.tombstones.Remove(op.ID)
	}
	if op.Side == spatial.XLocationLeft {
		origin.left = sortedInsert(origin.left, e)
	} else {
		origin.right = sortedInsert(origin.right, e)
	}
	t.elements[op.ID] = e
	t.dirty = true
	return e
}

// drain integrates buffered inserts whose origin has since arrived, repeating until no
// further progress is possible.
func (t *Text) drain() {
	for {
		progressed := false
		remaining := t.pending[:0]
		for _, op := range t.pending {
			if _, seen := t.elements[op.ID]; seen {
				continue
			}
			if _, hasOrigin := t.elements[op.Origin]; !isRoot(op.Origin) && !hasOrigin {
				remaining = append(remaining, op)
				continue
			}
			t.place(op)
			progressed = true
		}
		t.pending = remaining
		if !progressed {
			return
		}
	}
}

// sortedInsert inserts e into the id-sorted slice children and returns the result.
func sortedInsert(children []*element, e *element) []*element {
	lo, hi := 0, len(children)
	for lo < hi {
		mid := (lo + hi) / 2
		if idLess(children[mid].id, e.id) {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	children = append(children, nil)
	copy(children[lo+1:], children[lo:])
	children[lo] = e
	return children
}
