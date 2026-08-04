// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// This file is a frozen copy of the pre-run-length, one-element-per-character
// implementation. It is the reference model for the differential tests in
// differential_test.go; never edit it to track the production code.

package crdt_test

import (
	"slices"

	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/spatial"
)

// refIsRoot reports whether id is the document root sentinel. The zero replica is
// reserved for the root and never assigned to a real replica.
func refIsRoot(id crdt.ID) bool { return id.Replica == 0 && id.Counter == 0 }

// refIDLess reports whether a sorts before b, ordering by replica then counter. The
// order is the tiebreak between characters that share an anchor, so every replica
// resolves concurrent insertions identically.
func refIDLess(a, b crdt.ID) bool {
	if a.Replica != b.Replica {
		return a.Replica < b.Replica
	}
	return a.Counter < b.Counter
}

// refElement is a node in the document tree. The in-order traversal of the tree,
// skipping deleted nodes, yields the materialized document.
type refElement struct {
	id      crdt.ID
	char    int32
	deleted bool
	// left and right hold the children anchored on each side, each kept sorted by id.
	left  []*refElement
	right []*refElement
}

// Text is a replicated text document. A Text is owned by a single replica, identified
// at construction; local edits are attributed to that replica. Text is not safe for
// concurrent use.
type refText struct {
	replica  uint32
	counter  uint32
	root     *refElement
	elements map[crdt.ID]*refElement
	// tombstones holds ids deleted before their insert was seen, so the delete can be
	// applied to the character once it arrives.
	tombstones set.Set[crdt.ID]
	// pending holds inserts whose origin has not yet been integrated, buffered until
	// the
	// origin arrives.
	pending []crdt.Insert
	// order caches the in-order traversal of every refElement, rebuilt lazily when
	// dirty.
	order []*refElement
	dirty bool
	// visibleCache caches the live characters in document order. It is nil when stale
	// and
	// must be recomputed on the next read.
	visibleCache []*refElement
	// str caches the materialized string; strValid reports whether it is current.
	str      string
	strValid bool
	// live is the number of non-deleted characters, maintained incrementally so Len is
	// constant time.
	live int
}

// markDirty invalidates the cached traversal and derived caches after a mutation.
func (t *refText) markDirty() {
	t.dirty = true
	t.visibleCache = nil
	t.strValid = false
}

// New creates an empty document owned by the given replica. The replica must be
// non-zero and unique among the replicas editing the document.
func newRef(replica uint32) *refText {
	return &refText{
		replica:    replica,
		root:       &refElement{},
		elements:   make(map[crdt.ID]*refElement),
		tombstones: set.New[crdt.ID](),
	}
}

// Replica returns the id of the replica that owns this document.
func (t *refText) Replica() uint32 { return t.replica }

// Snapshot captures the full current state of the document as the operations that
// reconstruct it when applied to an empty document, in an order where every operation's
// origin precedes it. It is used to bootstrap a replica joining an in-progress session.
// Deleted characters are included as both an insert and a delete so anchoring is
// preserved.
func (t *refText) Snapshot() (inserts []crdt.Insert, deletes []crdt.Delete) {
	var walk func(e *refElement, origin crdt.ID, side spatial.XLocation)
	walk = func(e *refElement, origin crdt.ID, side spatial.XLocation) {
		if e != t.root {
			inserts = append(inserts,
				crdt.Insert{ID: e.id, Origin: origin, Side: side, Char: e.char})
			if e.deleted {
				deletes = append(deletes, crdt.Delete{ID: e.id})
			}
		}
		for _, c := range e.left {
			walk(c, e.id, spatial.XLocationLeft)
		}
		for _, c := range e.right {
			walk(c, e.id, spatial.XLocationRight)
		}
	}
	walk(t.root, crdt.ID{}, spatial.XLocationRight)
	return inserts, deletes
}

// Load applies a snapshot to the document. It is intended for a freshly created
// document; the local replica's edits remain attributed to its own replica id and do
// not collide with the snapshot's operations.
func (t *refText) Load(inserts []crdt.Insert, deletes []crdt.Delete) {
	t.ApplyInsert(inserts...)
	t.ApplyDelete(deletes...)
}

// rebuild regenerates the cached in-order traversal if it is stale.
func (t *refText) rebuild() {
	if !t.dirty && t.order != nil {
		return
	}
	t.order = t.order[:0]
	t.walk()
	t.dirty = false
}

// walk appends the in-order traversal of the tree to t.order, excluding the root
// sentinel. The traversal is iterative with an explicit stack because a document built
// by typing in a single direction forms a tree as deep as it is long, which would
// overflow the call stack under recursion.
func (t *refText) walk() {
	type frame struct {
		node *refElement
		emit bool
	}
	stack := []frame{{node: t.root}}
	for len(stack) > 0 {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if f.emit {
			t.order = append(t.order, f.node)
			continue
		}
		for _, r := range slices.Backward(f.node.right) {
			stack = append(stack, frame{node: r})
		}
		if f.node != t.root {
			stack = append(stack, frame{node: f.node, emit: true})
		}
		for _, l := range slices.Backward(f.node.left) {
			stack = append(stack, frame{node: l})
		}
	}
}

// visible returns the live characters in document order. The result is cached and
// reused until the next mutation.
func (t *refText) visible() []*refElement {
	if t.visibleCache != nil {
		return t.visibleCache
	}
	t.rebuild()
	out := make([]*refElement, 0, len(t.order))
	for _, e := range t.order {
		if !e.deleted {
			out = append(out, e)
		}
	}
	t.visibleCache = out
	return out
}

// Len returns the number of live characters in the document.
func (t *refText) Len() int { return t.live }

// Collectable returns the ids of the tombstoned characters that may be dropped from the
// document without changing its value or orphaning a surviving character: those whose
// entire subtree is also tombstoned, so no live character anchors to them, directly or
// transitively. A deleted character that still anchors a live character is load-bearing
// and is not returned. The result is only safe to remove once the document is also
// causally stable, since a concurrent edit may still anchor to a tombstone.
func (t *refText) Collectable() []crdt.ID {
	var out []crdt.ID
	var visit func(e *refElement) bool
	visit = func(e *refElement) bool {
		subtreeDead := true
		for _, c := range e.left {
			if !visit(c) {
				subtreeDead = false
			}
		}
		for _, c := range e.right {
			if !visit(c) {
				subtreeDead = false
			}
		}
		if e == t.root || !e.deleted || !subtreeDead {
			return false
		}
		out = append(out, e.id)
		return true
	}
	visit(t.root)
	return out
}

// String materializes the document into its current string value. The result is cached
// and reused until the next mutation.
func (t *refText) String() string {
	if t.strValid {
		return t.str
	}
	vis := t.visible()
	runes := make([]rune, len(vis))
	for i, e := range vis {
		runes[i] = rune(e.char)
	}
	t.str = string(runes)
	t.strValid = true
	return t.str
}

// IndexToID returns the id of the character at the given live index. The second return
// is false if the index is out of range.
func (t *refText) IndexToID(index int) (crdt.ID, bool) {
	vis := t.visible()
	if index < 0 || index >= len(vis) {
		return crdt.ID{}, false
	}
	return vis[index].id, true
}

// Insert inserts text at the given live index and returns the operations that
// describe the edit. The operations are applied to this document before they are
// returned, and must be broadcast to other replicas for them to converge. index is
// measured in code points and may equal Len to append.
func (t *refText) Insert(index int, text string) []crdt.Insert {
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}
	vis := t.visible()
	var left *refElement
	if index <= 0 {
		left = t.root
	} else if index-1 < len(vis) {
		left = vis[index-1]
	} else if len(vis) > 0 {
		left = vis[len(vis)-1]
	} else {
		left = t.root
	}
	var right *refElement
	if index >= 0 && index < len(vis) {
		right = vis[index]
	}
	ops := make([]crdt.Insert, 0, len(runes))
	for _, r := range runes {
		var origin crdt.ID
		var side spatial.XLocation
		if right != nil && len(left.right) > 0 {
			origin, side = right.id, spatial.XLocationLeft
		} else {
			origin, side = left.id, spatial.XLocationRight
		}
		t.counter++
		op := crdt.Insert{
			ID:     crdt.ID{Replica: t.replica, Counter: t.counter},
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
func (t *refText) Delete(index, length int) []crdt.Delete {
	if length <= 0 {
		return nil
	}
	vis := t.visible()
	ops := make([]crdt.Delete, 0, length)
	for i := index; i < index+length && i < len(vis); i++ {
		op := crdt.Delete{ID: vis[i].id}
		t.ApplyDelete(op)
		ops = append(ops, op)
	}
	return ops
}

// ApplyInsert integrates insert operations produced by other replicas. Operations may
// be supplied in any order and duplicates are ignored; operations whose origin has not
// yet arrived are buffered and integrated once it does.
func (t *refText) ApplyInsert(ops ...crdt.Insert) {
	for _, op := range ops {
		if t.place(op) != nil {
			t.drain()
		} else {
			t.pending = append(t.pending, op)
		}
	}
}

// ApplyDelete integrates delete operations produced by other replicas. A delete whose
// character has not yet been seen is recorded so the character is tombstoned on
// arrival.
func (t *refText) ApplyDelete(ops ...crdt.Delete) {
	for _, op := range ops {
		if e, ok := t.elements[op.ID]; ok {
			if !e.deleted {
				e.deleted = true
				t.live--
				t.markDirty()
			}
			continue
		}
		t.tombstones.Add(op.ID)
	}
}

// place attaches an insert to the tree, returning the created refElement. It returns
// nil without buffering when the operation was already integrated or its origin is not
// yet present.
func (t *refText) place(op crdt.Insert) *refElement {
	if e, ok := t.elements[op.ID]; ok {
		return e
	}
	origin := t.root
	if !refIsRoot(op.Origin) {
		var ok bool
		if origin, ok = t.elements[op.Origin]; !ok {
			return nil
		}
	}
	e := &refElement{id: op.ID, char: op.Char}
	if t.tombstones.Contains(op.ID) {
		e.deleted = true
		t.tombstones.Remove(op.ID)
	}
	if op.Side == spatial.XLocationLeft {
		origin.left = refSortedInsert(origin.left, e)
	} else {
		origin.right = refSortedInsert(origin.right, e)
	}
	t.elements[op.ID] = e
	if !e.deleted {
		t.live++
	}
	t.markDirty()
	return e
}

// drain integrates buffered inserts whose origin has since arrived, repeating until no
// further progress is possible.
func (t *refText) drain() {
	for {
		progressed := false
		remaining := t.pending[:0]
		for _, op := range t.pending {
			if _, seen := t.elements[op.ID]; seen {
				continue
			}
			if _, hasOrigin := t.elements[op.Origin]; !refIsRoot(op.Origin) &&
				!hasOrigin {
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

// refSortedInsert inserts e into the id-sorted slice children and returns the result.
func refSortedInsert(children []*refElement, e *refElement) []*refElement {
	lo, hi := 0, len(children)
	for lo < hi {
		mid := (lo + hi) / 2
		if refIDLess(children[mid].id, e.id) {
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
