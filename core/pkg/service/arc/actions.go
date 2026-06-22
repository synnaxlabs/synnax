// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"maps"
	"sync"

	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// Handle replaces the Arc module's name.
func (p RenamePayload) Handle(state Arc) (Arc, error) {
	state.Name = p.Name
	return state, nil
}

// Handle switches the module between text and graph representation.
func (p SetModePayload) Handle(state Arc) (Arc, error) {
	state.Mode = p.Mode
	return state, nil
}

// Handle inserts the node if no node with the same key exists, otherwise
// replaces the existing node in place.
func (p SetNodePayload) Handle(state Arc) (Arc, error) {
	for i := range state.Graph.Nodes {
		if state.Graph.Nodes[i].Key == p.Node.Key {
			state.Graph.Nodes[i] = p.Node
			return state, nil
		}
	}
	state.Graph.Nodes = append(state.Graph.Nodes, p.Node)
	return state, nil
}

// Handle moves the named node to the given canvas position. No-op if no node
// matches.
func (p SetNodePositionPayload) Handle(state Arc) (Arc, error) {
	for i := range state.Graph.Nodes {
		if state.Graph.Nodes[i].Key == p.Key {
			state.Graph.Nodes[i].Position = p.Position
			break
		}
	}
	return state, nil
}

// Handle merges the payload config into the configs entry for the given key in
// the graph configs map. Top-level fields present in the payload overwrite
// existing fields; fields absent from the payload are preserved.
func (p SetNodeConfigPayload) Handle(state Arc) (Arc, error) {
	if state.Graph.Configs == nil {
		state.Graph.Configs = make(map[string]msgpack.EncodedJSON)
	}
	if existing := state.Graph.Configs[p.Key]; existing != nil {
		merged := make(msgpack.EncodedJSON, len(existing)+len(p.Config))
		maps.Copy(merged, existing)
		maps.Copy(merged, p.Config)
		state.Graph.Configs[p.Key] = merged
		return state, nil
	}
	state.Graph.Configs[p.Key] = p.Config
	return state, nil
}

// Handle removes the node with the matching key along with any edges connected
// to it.
func (p RemoveNodePayload) Handle(state Arc) (Arc, error) {
	for i := range state.Graph.Nodes {
		if state.Graph.Nodes[i].Key == p.Key {
			state.Graph.Nodes = append(state.Graph.Nodes[:i], state.Graph.Nodes[i+1:]...)
			break
		}
	}
	delete(state.Graph.Configs, p.Key)
	kept := make(graph.Edges, 0, len(state.Graph.Edges))
	for _, e := range state.Graph.Edges {
		if e.Source.Node == p.Key || e.Target.Node == p.Key {
			continue
		}
		kept = append(kept, e)
	}
	state.Graph.Edges = kept
	return state, nil
}

// Handle appends the character insertion to the arc's replicated text document. The
// document is an op-log; materialization (see text.Text.Materialize) deduplicates and
// orders, so applying an operation is appending it.
func (p InsertCharPayload) Handle(state Arc) (Arc, error) {
	state.Text.Doc.Inserts = append(
		state.Text.Doc.Inserts,
		crdt.Insert{ID: p.ID, Origin: p.Origin, Side: p.Side, Char: p.Char},
	)
	return state, nil
}

// Handle appends the edge to the graph. No-op when an edge with the same source
// and target handles already exists, so concurrent additions of the same
// connection converge regardless of differing keys.
func (p AddEdgePayload) Handle(state Arc) (Arc, error) {
	for _, e := range state.Graph.Edges {
		if e.Source == p.Edge.Source && e.Target == p.Edge.Target {
			return state, nil
		}
	}
	state.Graph.Edges = append(state.Graph.Edges, p.Edge)
	return state, nil
}

// Handle removes the edge with the given key, if present.
func (p RemoveEdgePayload) Handle(state Arc) (Arc, error) {
	kept := make(graph.Edges, 0, len(state.Graph.Edges))
	for _, e := range state.Graph.Edges {
		if e.Key == p.Key {
			continue
		}
		kept = append(kept, e)
	}
	state.Graph.Edges = kept
	return state, nil
}

// Handle rewrites the endpoints of the edge with the given key, preserving its
// key and kind. No-op when no edge with the key exists.
func (p ReconnectEdgePayload) Handle(state Arc) (Arc, error) {
	for i := range state.Graph.Edges {
		if state.Graph.Edges[i].Key == p.Key {
			state.Graph.Edges[i].Source = p.Source
			state.Graph.Edges[i].Target = p.Target
			break
		}
	}
	return state, nil
}

// Handle appends the character deletion to the arc's replicated text document. See
// InsertCharPayload.Handle.
func (p DeleteCharPayload) Handle(state Arc) (Arc, error) {
	state.Text.Doc.Deletes = append(state.Text.Doc.Deletes, crdt.Delete{ID: p.ID})
	return state, nil
}

// Handle drops the insert and delete operations of the given already-deleted characters
// from the arc's replicated text document. Because the characters are tombstoned, and so
// invisible, removing their operations does not change the materialized text. It is
// emitted by the server's text sweeper to reclaim the space held by tombstones.
func (p ForgetCharsPayload) Handle(state Arc) (Arc, error) {
	if len(p.IDs) == 0 {
		return state, nil
	}
	forget := set.New(p.IDs...)
	doc := state.Text.Doc
	keptInserts := make([]crdt.Insert, 0, len(doc.Inserts))
	for _, in := range doc.Inserts {
		if !forget.Contains(in.ID) {
			keptInserts = append(keptInserts, in)
		}
	}
	keptDeletes := make([]crdt.Delete, 0, len(doc.Deletes))
	for _, del := range doc.Deletes {
		if !forget.Contains(del.ID) {
			keptDeletes = append(keptDeletes, del)
		}
	}
	state.Text.Doc.Inserts = keptInserts
	state.Text.Doc.Deletes = keptDeletes
	return state, nil
}

const (
	// defaultTextSweepQuiescence is how long an arc's text must go unedited before its
	// tombstoned characters become eligible to be forgotten.
	defaultTextSweepQuiescence = 5 * telem.Second
	// defaultTextSweepThreshold is the number of tombstoned characters that must
	// accumulate before a sweep is worth broadcasting.
	defaultTextSweepThreshold = 128
)

// lastEdits records the cluster time of each arc's most recent character edit. The
// timestamp only gates the text sweeper's quiescence check; it never replicates and
// need not survive a node restart, so it is held in memory rather than persisted on the
// arc. A restarted node reads the zero time for every arc, which is safe: a zero last
// edit reads as quiet, and a quiet arc is exactly one whose tombstones are eligible to
// be forgotten. lastEdits is safe for concurrent use.
type lastEdits struct {
	mu    sync.Mutex
	times map[Key]telem.TimeStamp
}

func newLastEdits() *lastEdits {
	return &lastEdits{times: make(map[Key]telem.TimeStamp)}
}

// get returns the recorded last-edit time for the arc, or the zero time if none has been
// recorded.
func (l *lastEdits) get(key Key) telem.TimeStamp {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.times[key]
}

// set records ts as the arc's most recent character-edit time.
func (l *lastEdits) set(key Key, ts telem.TimeStamp) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.times[key] = ts
}

// forget drops the recorded last-edit time for the arc. It is called when the arc is
// deleted so the map does not retain keys for arcs that no longer exist.
func (l *lastEdits) forget(key Key) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.times, key)
}

// textSweeper reclaims the space held by tombstoned characters in an arc's replicated
// text document. Deleting a character leaves both its insert and delete operation in the
// op-log forever, so a heavily edited document grows without bound; the sweeper drops
// those operations once it is safe to do so.
//
// Safety hinges on quiescence. A tombstoned character can only be forgotten once every
// editor has observed its deletion: an editor that still sees the character could anchor
// a new insertion to it, and that insertion would be orphaned if the character were
// already gone. The sweeper treats an arc as safe to sweep when no edit has reached it
// for the quiescence window, by which point every outstanding delete has propagated to
// every editor.
type textSweeper struct {
	now        func() telem.TimeStamp
	quiescence telem.TimeSpan
	threshold  int
	// edits tracks the most recent character-edit time per arc, gating the quiescence
	// check below.
	edits *lastEdits
}

func newTextSweeper(
	now func() telem.TimeStamp,
	quiescence telem.TimeSpan,
	threshold int,
) textSweeper {
	return textSweeper{
		now:        now,
		quiescence: quiescence,
		threshold:  threshold,
		edits:      newLastEdits(),
	}
}

// quiet reports whether enough time has elapsed since the arc's most recent character
// edit that its tombstoned characters are safe to forget.
func (s textSweeper) quiet(key Key) bool {
	return s.edits.get(key).Span(s.now()) > s.quiescence
}

// recordEdit marks the current time as the arc's most recent character edit, restarting
// its quiescence window.
func (s textSweeper) recordEdit(key Key) { s.edits.set(key, s.now()) }

// forget discards the arc's tracked edit time. It is called when the arc is deleted so
// the tracker does not retain keys for arcs that no longer exist.
func (s textSweeper) forget(key Key) { s.edits.forget(key) }

// forgettable returns the ids of the tombstoned characters in doc that are safe to drop:
// those whose entire subtree is also tombstoned, so removing them orphans no surviving
// character (see crdt.Text.Collectable). It returns nil when fewer than the configured
// threshold are present, so a sweep is only triggered once enough space is reclaimable to
// justify the broadcast.
func (s textSweeper) forgettable(doc text.Document) []crdt.ID {
	if len(doc.Deletes) == 0 {
		return nil
	}
	t := crdt.New(text.SeedReplica)
	t.Load(doc.Inserts, doc.Deletes)
	dead := t.Collectable()
	if len(dead) < s.threshold {
		return nil
	}
	return dead
}

// containsTextEdit reports whether actions includes a character insertion or deletion,
// the edits that restart an arc's quiescence window.
func containsTextEdit(actions []Action) bool {
	for _, a := range actions {
		if a.Type == ActionTypeInsertChar || a.Type == ActionTypeDeleteChar {
			return true
		}
	}
	return false
}
