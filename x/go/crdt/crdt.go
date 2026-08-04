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
// value. The merge rule is the non-interleaving list CRDT in which each character is
// anchored to an existing character on its left or right, so that concurrent runs of
// text inserted at the same position are never interleaved. Contiguous characters
// authored by one replica are stored as a single run node, split only when a
// concurrent edit anchors inside the run.
//
// The Insert, Delete, and ID types are generated from schemas/crdt.oracle and shared
// with the TypeScript implementation (@synnaxlabs/x crdt); the two runtimes materialize
// identically, locked by the conformance vectors in testdata.
package crdt

import (
	"slices"

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

// element is a run of characters authored by one replica with contiguous counters. The
// in-order traversal of the run tree, skipping deleted characters, yields the
// materialized document. Invariants: left children anchor to the first character, right
// children anchor to the last, and interior characters never carry children; an anchor
// into the interior splits the run first.
type element struct {
	// id identifies the first character; the character at offset i has counter
	// id.Counter+i.
	id    ID
	chars []int32
	// deleted marks tombstoned characters. It is nil when none are; otherwise its
	// length always equals len(chars).
	deleted []bool
	// dead is the number of tombstoned characters in this run.
	dead int
	// left and right hold the children anchored on each side, each kept sorted by id.
	left  []*element
	right []*element
}

// lastID returns the id of the run's last character.
func (e *element) lastID() ID {
	return ID{Replica: e.id.Replica, Counter: e.id.Counter + uint32(len(e.chars)-1)}
}

// charID returns the id of the character at offset i.
func (e *element) charID(i int) ID {
	return ID{Replica: e.id.Replica, Counter: e.id.Counter + uint32(i)}
}

// kill tombstones the character at offset off, reporting whether it was live.
func (e *element) kill(off int) bool {
	if e.deleted == nil {
		e.deleted = make([]bool, len(e.chars))
	}
	if e.deleted[off] {
		return false
	}
	e.deleted[off] = true
	e.dead++
	return true
}

// Text is a replicated text document. A Text is owned by a single replica, identified
// at construction; local edits are attributed to that replica. Text is not safe for
// concurrent use.
type Text struct {
	replica uint32
	counter uint32
	root    *element
	// index maps each replica to its runs ordered by starting counter, so any character
	// id resolves to its containing run by binary search.
	index map[uint32][]*element
	// tombstones holds ids deleted before their insert was seen, so the delete can be
	// applied to the character once it arrives.
	tombstones set.Set[ID]
	// pending holds inserts whose origin has not yet been integrated, buffered until the
	// origin arrives.
	pending []Insert
	// order caches the in-order traversal of every run, rebuilt lazily when dirty.
	order []*element
	dirty bool
	// lastPlaced is the run most recently created or extended: the likely target of the
	// next sequential insert, letting a typed or seeded run extend without an id lookup.
	lastPlaced *element
	// str caches the materialized string; strValid reports whether it is current.
	str      string
	strValid bool
	// live is the number of non-deleted characters, maintained incrementally so Len is
	// constant time.
	live int
}

// markDirty invalidates every derived cache after a structural mutation: one that
// creates, splits, or attaches a run and so changes the traversal.
func (t *Text) markDirty() {
	t.dirty = true
	t.strValid = false
}

// markStale invalidates only the materialized string, for mutations that change a
// run's content in place (an appended or tombstoned character) without changing the
// traversal.
func (t *Text) markStale() { t.strValid = false }

// New creates an empty document owned by the given replica. The replica must be non-zero
// and unique among the replicas editing the document.
func New(replica uint32) *Text {
	return &Text{
		replica:    replica,
		root:       &element{},
		index:      make(map[uint32][]*element),
		tombstones: set.New[ID](),
	}
}

// Replica returns the id of the replica that owns this document.
func (t *Text) Replica() uint32 { return t.replica }

// findRun returns the run containing id and the character's offset within it.
func (t *Text) findRun(id ID) (*element, int, bool) {
	runs := t.index[id.Replica]
	lo, hi := 0, len(runs)
	for lo < hi {
		mid := (lo + hi) / 2
		if runs[mid].id.Counter <= id.Counter {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo == 0 {
		return nil, 0, false
	}
	e := runs[lo-1]
	off := int(id.Counter - e.id.Counter)
	if off < len(e.chars) {
		return e, off, true
	}
	return nil, 0, false
}

// indexInsert records a new run in the replica index, keeping runs counter-ordered.
func (t *Text) indexInsert(e *element) {
	runs := t.index[e.id.Replica]
	lo, hi := 0, len(runs)
	for lo < hi {
		mid := (lo + hi) / 2
		if runs[mid].id.Counter < e.id.Counter {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	runs = append(runs, nil)
	copy(runs[lo+1:], runs[lo:])
	runs[lo] = e
	t.index[e.id.Replica] = runs
}

// splitAfter splits e after character offset k (0 <= k < len-1) and returns the new
// tail run. The tail inherits e's right children, since they anchor to e's old last
// character, and becomes e's sole right child.
func (t *Text) splitAfter(e *element, k int) *element {
	tail := &element{id: e.charID(k + 1), chars: e.chars[k+1:]}
	e.chars = e.chars[: k+1 : k+1]
	if e.deleted != nil {
		tail.deleted = e.deleted[k+1:]
		e.deleted = e.deleted[: k+1 : k+1]
		for _, d := range tail.deleted {
			if d {
				tail.dead++
			}
		}
		e.dead -= tail.dead
	}
	tail.right = e.right
	e.right = []*element{tail}
	t.indexInsert(tail)
	return tail
}

// Snapshot captures the full current state of the document as the operations that
// reconstruct it when applied to an empty document, in an order where every operation's
// origin precedes it. It is used to bootstrap a replica joining an in-progress session.
// Deleted characters are included as both an insert and a delete so anchoring is
// preserved.
func (t *Text) Snapshot() (inserts []Insert, deletes []Delete) {
	var walk func(e *element, origin ID, side spatial.XLocation)
	walk = func(e *element, origin ID, side spatial.XLocation) {
		first, last := origin, origin
		if e != t.root {
			prev, prevSide := origin, side
			for i, c := range e.chars {
				id := e.charID(i)
				inserts = append(inserts, Insert{ID: id, Origin: prev, Side: prevSide, Char: c})
				if e.deleted != nil && e.deleted[i] {
					deletes = append(deletes, Delete{ID: id})
				}
				prev, prevSide = id, spatial.XLocationRight
			}
			first, last = e.id, e.lastID()
		}
		for _, c := range e.left {
			walk(c, first, spatial.XLocationLeft)
		}
		for _, c := range e.right {
			walk(c, last, spatial.XLocationRight)
		}
	}
	walk(t.root, ID{}, spatial.XLocationRight)
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
	t.walk()
	t.dirty = false
}

// walk appends the in-order traversal of the run tree to t.order, excluding the root
// sentinel. The traversal is iterative with an explicit stack because a heavily
// interleaved document forms a tree as deep as its run count, which could overflow the
// call stack under recursion.
func (t *Text) walk() {
	type frame struct {
		node *element
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

// charAt returns the run and character offset of the live character at index.
func (t *Text) charAt(index int) (*element, int, bool) {
	if index < 0 || index >= t.live {
		return nil, 0, false
	}
	t.rebuild()
	seen := 0
	for _, e := range t.order {
		liveHere := len(e.chars) - e.dead
		if index >= seen+liveHere {
			seen += liveHere
			continue
		}
		k := index - seen
		if e.dead == 0 {
			return e, k, true
		}
		for i := range e.chars {
			if e.deleted[i] {
				continue
			}
			if k == 0 {
				return e, i, true
			}
			k--
		}
	}
	return nil, 0, false
}

// Len returns the number of live characters in the document.
func (t *Text) Len() int { return t.live }

// Collectable returns the ids of the tombstoned characters that may be dropped from the
// document without changing its value or orphaning a surviving character: those whose
// entire subtree is also tombstoned, so no live character anchors to them, directly or
// transitively. A deleted character that still anchors a live character is load-bearing
// and is not returned. The result is only safe to remove once the document is also
// causally stable, since a concurrent edit may still anchor to a tombstone.
func (t *Text) Collectable() []ID {
	var out []ID
	var visit func(e *element) bool
	visit = func(e *element) bool {
		leftDead, rightDead := true, true
		for _, c := range e.left {
			if !visit(c) {
				leftDead = false
			}
		}
		for _, c := range e.right {
			if !visit(c) {
				rightDead = false
			}
		}
		if e == t.root || !rightDead {
			return false
		}
		n := len(e.chars)
		// Only the fully-tombstoned tail of a run can be collectable, since each
		// character anchors its successor.
		sufStart := n
		if e.deleted != nil {
			for i := n - 1; i >= 0 && e.deleted[i]; i-- {
				sufStart = i
			}
		}
		for i := sufStart; i < n; i++ {
			// The first character also anchors the run's left children.
			if i == 0 && !leftDead {
				continue
			}
			out = append(out, e.charID(i))
		}
		return e.dead == n && leftDead && rightDead
	}
	visit(t.root)
	return out
}

// String materializes the document into its current string value. The result is cached
// and reused until the next mutation.
func (t *Text) String() string {
	if t.strValid {
		return t.str
	}
	t.rebuild()
	runes := make([]rune, 0, t.live)
	for _, e := range t.order {
		if e.dead == 0 {
			for _, c := range e.chars {
				runes = append(runes, rune(c))
			}
			continue
		}
		for i, c := range e.chars {
			if !e.deleted[i] {
				runes = append(runes, rune(c))
			}
		}
	}
	t.str = string(runes)
	t.strValid = true
	return t.str
}

// IndexToID returns the id of the character at the given live index. The second return
// is false if the index is out of range.
func (t *Text) IndexToID(index int) (ID, bool) {
	e, off, ok := t.charAt(index)
	if !ok {
		return ID{}, false
	}
	return e.charID(off), true
}

// hasRight reports whether the character at offset off anchors anything on its right: a
// successor within the run, or, for the last character, a right child.
func hasRight(e *element, off int) bool {
	if off < len(e.chars)-1 {
		return true
	}
	return len(e.right) > 0
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
	left, leftOff := t.root, -1
	if index > 0 {
		if e, off, ok := t.charAt(index - 1); ok {
			left, leftOff = e, off
		} else if e, off, ok := t.charAt(t.live - 1); ok {
			left, leftOff = e, off
		}
	}
	var rightID ID
	haveRight := false
	if e, off, ok := t.charAt(index); ok {
		rightID, haveRight = e.charID(off), true
	}
	ops := make([]Insert, 0, len(runes))
	for _, r := range runes {
		var origin ID
		var side spatial.XLocation
		switch {
		case haveRight && hasRight(left, leftOff):
			origin, side = rightID, spatial.XLocationLeft
		case left == t.root:
			origin, side = ID{}, spatial.XLocationRight
		default:
			origin, side = left.charID(leftOff), spatial.XLocationRight
		}
		t.counter++
		op := Insert{
			ID:     ID{Replica: t.replica, Counter: t.counter},
			Origin: origin,
			Side:   side,
			Char:   r,
		}
		left, leftOff = t.place(op)
		ops = append(ops, op)
	}
	return ops
}

// Delete removes length characters starting at the given live index and returns the
// operations that describe the edit. The operations are applied to this document before
// they are returned.
func (t *Text) Delete(index, length int) []Delete {
	if length <= 0 || index < 0 || index >= t.live {
		return nil
	}
	t.rebuild()
	end := min(index+length, t.live)
	ids := make([]ID, 0, end-index)
	seen := 0
	for _, e := range t.order {
		liveHere := len(e.chars) - e.dead
		if seen+liveHere <= index {
			seen += liveHere
			continue
		}
		for i := range e.chars {
			if e.deleted != nil && e.deleted[i] {
				continue
			}
			if seen >= index && seen < end {
				ids = append(ids, e.charID(i))
			}
			seen++
		}
		if seen >= end {
			break
		}
	}
	ops := make([]Delete, 0, len(ids))
	for _, id := range ids {
		op := Delete{ID: id}
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
		if e, _ := t.place(op); e != nil {
			if len(t.pending) > 0 {
				t.drain()
			}
		} else {
			t.pending = append(t.pending, op)
		}
	}
}

// ApplyDelete integrates delete operations produced by other replicas. A delete whose
// character has not yet been seen is recorded so the character is tombstoned on arrival.
func (t *Text) ApplyDelete(ops ...Delete) {
	for _, op := range ops {
		if e, off, ok := t.findRun(op.ID); ok {
			if e.kill(off) {
				t.live--
				t.markStale()
			}
			continue
		}
		t.tombstones.Add(op.ID)
	}
}

// extendable returns the run op contiguously appends to, or nil: the run last touched
// by place, when op's origin is its last character, replica and counter continue it,
// and nothing anchors after it. Such an op cannot be a duplicate: were its counter
// already placed, the run would have been split or given a right child, and the check
// would fail.
func (t *Text) extendable(op Insert) *element {
	e := t.lastPlaced
	if e != nil && op.Side == spatial.XLocationRight &&
		len(e.right) == 0 &&
		op.ID.Replica == e.id.Replica &&
		op.Origin == e.lastID() &&
		op.ID.Counter == e.id.Counter+uint32(len(e.chars)) {
		return e
	}
	return nil
}

// extend appends op's character to the run extendable approved.
func (t *Text) extend(e *element, op Insert) (*element, int) {
	e.chars = append(e.chars, op.Char)
	if e.deleted != nil {
		e.deleted = append(e.deleted, false)
	}
	off := len(e.chars) - 1
	if len(t.tombstones) > 0 && t.tombstones.Contains(op.ID) {
		e.kill(off)
		t.tombstones.Remove(op.ID)
	} else {
		t.live++
	}
	t.markStale()
	return e, off
}

// place attaches an insert to the run tree, returning the run and offset holding the
// character. It returns nil without buffering when the operation's origin is not yet
// present.
func (t *Text) place(op Insert) (*element, int) {
	if e := t.extendable(op); e != nil {
		return t.extend(e, op)
	}
	if e, off, ok := t.findRun(op.ID); ok {
		return e, off
	}
	origin, originOff := t.root, -1
	if !isRoot(op.Origin) {
		var ok bool
		if origin, originOff, ok = t.findRun(op.Origin); !ok {
			return nil, 0
		}
	}
	e := &element{id: op.ID, chars: []int32{op.Char}}
	if t.tombstones.Contains(op.ID) {
		e.kill(0)
		t.tombstones.Remove(op.ID)
	} else {
		t.live++
	}
	if op.Side == spatial.XLocationLeft {
		if originOff > 0 {
			origin = t.splitAfter(origin, originOff-1)
		}
		origin.left = sortedInsert(origin.left, e)
	} else {
		if originOff >= 0 && originOff < len(origin.chars)-1 {
			t.splitAfter(origin, originOff)
		}
		origin.right = sortedInsert(origin.right, e)
	}
	t.indexInsert(e)
	t.lastPlaced = e
	t.markDirty()
	return e, 0
}

// drain integrates buffered inserts whose origin has since arrived, repeating until no
// further progress is possible.
func (t *Text) drain() {
	for {
		progressed := false
		remaining := t.pending[:0]
		for _, op := range t.pending {
			if _, _, seen := t.findRun(op.ID); seen {
				continue
			}
			if _, _, hasOrigin := t.findRun(op.Origin); !isRoot(op.Origin) && !hasOrigin {
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
