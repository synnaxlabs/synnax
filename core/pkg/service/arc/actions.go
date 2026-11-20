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
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates an
// Arc.
func (CreatePayload) Handle(Arc) (Arc, error) {
	return Arc{}, actions.CoreOnlyError("an Arc create")
}

// Handle inserts the node if no node with the same key exists, otherwise
// replaces the existing node in place.
func (p SetNodePayload) Handle(state Arc) (Arc, error) {
	return onGraph(state, func(g *graph.Graph) {
		for i := range g.Nodes {
			if g.Nodes[i].Key == p.Node.Key {
				g.Nodes[i] = p.Node
				return
			}
		}
		g.Nodes = append(g.Nodes, p.Node)
	})
}

// Handle moves the named node to the given canvas position. No-op if no node
// matches.
func (p SetNodePositionPayload) Handle(state Arc) (Arc, error) {
	return onGraph(state, func(g *graph.Graph) {
		for i := range g.Nodes {
			if g.Nodes[i].Key == p.Key {
				g.Nodes[i].Position = p.Position
				return
			}
		}
	})
}

// Handle merges the payload inputs into the entry for the given key in
// the graph inputs map. Top-level fields present in the payload overwrite
// existing fields; fields absent from the payload are preserved.
func (p SetNodeInputsPayload) Handle(state Arc) (Arc, error) {
	return onGraph(state, func(g *graph.Graph) {
		if g.Inputs == nil {
			g.Inputs = make(map[string]msgpack.EncodedJSON)
		}
		existing := g.Inputs[p.Key]
		if existing == nil {
			g.Inputs[p.Key] = p.Inputs
			return
		}
		merged := make(msgpack.EncodedJSON, len(existing)+len(p.Inputs))
		maps.Copy(merged, existing)
		maps.Copy(merged, p.Inputs)
		g.Inputs[p.Key] = merged
	})
}

// Handle removes the node with the matching key along with any edges connected
// to it.
func (p RemoveNodePayload) Handle(state Arc) (Arc, error) {
	return onGraph(state, func(g *graph.Graph) {
		for i := range g.Nodes {
			if g.Nodes[i].Key == p.Key {
				g.Nodes = append(g.Nodes[:i], g.Nodes[i+1:]...)
				break
			}
		}
		delete(g.Inputs, p.Key)
		kept := make(graph.Edges, 0, len(g.Edges))
		for _, e := range g.Edges {
			if e.Source.Node == p.Key || e.Target.Node == p.Key {
				continue
			}
			kept = append(kept, e)
		}
		g.Edges = kept
	})
}

// Handle appends the character insertion to the Arc's replicated text document. The
// document is an op-log; materialization (see text.Text.Materialize) deduplicates and
// orders, so applying an operation is appending it.
func (p InsertCharPayload) Handle(state Arc) (Arc, error) {
	return onText(state, func(t *text.Text) {
		t.Doc.Inserts = append(
			t.Doc.Inserts,
			crdt.Insert{ID: p.ID, Origin: p.Origin, Side: p.Side, Char: p.Char},
		)
	})
}

// Handle appends the edge to the graph. No-op when an edge with the same source
// and target handles already exists, so concurrent additions of the same
// connection converge regardless of differing keys.
func (p AddEdgePayload) Handle(state Arc) (Arc, error) {
	return onGraph(state, func(g *graph.Graph) {
		for _, e := range g.Edges {
			if e.Source == p.Edge.Source && e.Target == p.Edge.Target {
				return
			}
		}
		g.Edges = append(g.Edges, p.Edge)
	})
}

// Handle removes the edge with the given key, if present.
func (p RemoveEdgePayload) Handle(state Arc) (Arc, error) {
	return onGraph(state, func(g *graph.Graph) {
		kept := make(graph.Edges, 0, len(g.Edges))
		for _, e := range g.Edges {
			if e.Key != p.Key {
				kept = append(kept, e)
			}
		}
		g.Edges = kept
	})
}

// Handle rewrites the endpoints of the edge with the given key, preserving its
// key and kind. No-op when no edge with the key exists.
func (p ReconnectEdgePayload) Handle(state Arc) (Arc, error) {
	return onGraph(state, func(g *graph.Graph) {
		for i := range g.Edges {
			if g.Edges[i].Key == p.Key {
				g.Edges[i].Source = p.Source
				g.Edges[i].Target = p.Target
				return
			}
		}
	})
}

// Handle appends the character deletion to the Arc's replicated text document. See
// InsertCharPayload.Handle.
func (p DeleteCharPayload) Handle(state Arc) (Arc, error) {
	return onText(state, func(t *text.Text) {
		t.Doc.Deletes = append(t.Doc.Deletes, crdt.Delete{ID: p.ID})
	})
}

// Handle drops the insert and delete operations of the given already-deleted characters
// from the Arc's replicated text document. Because the characters are tombstoned, and
// so invisible, removing their operations does not change the materialized text. It is
// emitted by the server's text sweeper to reclaim the space held by tombstones.
func (p ForgetCharsPayload) Handle(state Arc) (Arc, error) {
	if len(p.IDs) == 0 {
		return state, nil
	}
	forget := set.New(p.IDs...)
	return onText(state, func(t *text.Text) {
		keptInserts := make([]crdt.Insert, 0, len(t.Doc.Inserts))
		for _, in := range t.Doc.Inserts {
			if !forget.Contains(in.ID) {
				keptInserts = append(keptInserts, in)
			}
		}
		keptDeletes := make([]crdt.Delete, 0, len(t.Doc.Deletes))
		for _, del := range t.Doc.Deletes {
			if !forget.Contains(del.ID) {
				keptDeletes = append(keptDeletes, del)
			}
		}
		t.Doc.Inserts = keptInserts
		t.Doc.Deletes = keptDeletes
	})
}

// Handle sets the directory that holds the Arc. Writer.Apply checks the Arc's name in
// that directory.
func (p SetDirectoryPayload) Handle(state Arc) (Arc, error) {
	b := state.Base()
	b.Directory = &p.Directory
	state.SetBase(b)
	return state, nil
}

// onGraph applies edit to the graph of a graph-mode Arc. A text-mode Arc is a
// validation error.
func onGraph(state Arc, edit func(g *graph.Graph)) (Arc, error) {
	b := state.Base()
	src, ok := b.Source.Variant.(GraphSource)
	if !ok {
		return Arc{}, errWrongMode(b, ModeGraph)
	}
	edit(&src.Graph)
	b.Source.Variant = src
	state.SetBase(b)
	return state, nil
}

// onText applies edit to the text of a text-mode Arc. A graph-mode Arc is a validation
// error.
func onText(state Arc, edit func(t *text.Text)) (Arc, error) {
	b := state.Base()
	src, ok := b.Source.Variant.(TextSource)
	if !ok {
		return Arc{}, errWrongMode(b, ModeText)
	}
	edit(&src.Text)
	b.Source.Variant = src
	state.SetBase(b)
	return state, nil
}

func errWrongMode(b Base, want Mode) error {
	return errors.Wrapf(
		validate.ErrValidation,
		"%q is a %s Arc and cannot take a %s edit",
		b.Name,
		b.Source.Mode(),
		want,
	)
}

// lastEdits records the cluster time of each Arc's most recent character edit. The
// timestamp only gates the text sweeper's quiescence check; it never replicates and
// need not survive a node restart, so it is held in memory rather than persisted on the
// Arc. A restarted node reads the zero time for every Arc, which is safe: a zero last
// edit reads as quiet, and a quiet Arc is exactly one whose tombstones are eligible to
// be forgotten. lastEdits is safe for concurrent use.
type lastEdits struct {
	mu    sync.Mutex
	times map[Key]telem.TimeStamp
}

func newLastEdits() *lastEdits {
	return &lastEdits{times: make(map[Key]telem.TimeStamp)}
}

// get returns the recorded last-edit time for the Arc, or the zero time if none has
// been recorded.
func (l *lastEdits) get(key Key) telem.TimeStamp {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.times[key]
}

// set records ts as the Arc's most recent character-edit time.
func (l *lastEdits) set(key Key, ts telem.TimeStamp) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.times[key] = ts
}

// forget drops the recorded last-edit time for the Arc. It is called when the Arc is
// deleted so the map does not retain keys for arcs that no longer exist.
func (l *lastEdits) forget(key Key) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.times, key)
}

// textSweeper reclaims the space held by tombstoned characters in an Arc's replicated
// text document. Deleting a character leaves both its insert and delete operation in
// the op-log forever, so a heavily edited document grows without bound; the sweeper
// drops those operations once it is safe to do so.
//
// Safety hinges on quiescence. A tombstoned character can only be forgotten once every
// editor has observed its deletion: an editor that still sees the character could
// anchor a new insertion to it, and that insertion would be orphaned if the character
// were already gone. The sweeper treats an Arc as safe to sweep when no edit has
// reached it for the quiescence window, by which point every outstanding delete has
// propagated to every editor.
type textSweeper struct {
	now        func() telem.TimeStamp
	quiescence telem.TimeSpan
	threshold  int
	// edits tracks the most recent character-edit time per Arc, gating the quiescence
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

// quiet reports whether enough time has elapsed since the Arc's most recent character
// edit that its tombstoned characters are safe to forget.
func (s textSweeper) quiet(key Key) bool {
	return s.edits.get(key).Span(s.now()) > s.quiescence
}

// recordEdit marks the current time as the Arc's most recent character edit, restarting
// its quiescence window.
func (s textSweeper) recordEdit(key Key) { s.edits.set(key, s.now()) }

// forget discards the Arc's tracked edit time. It is called when the Arc is deleted so
// the tracker does not retain keys for arcs that no longer exist.
func (s textSweeper) forget(key Key) { s.edits.forget(key) }

// forgettable returns the ids of the tombstoned characters in doc that are safe to
// drop: those whose entire subtree is also tombstoned, so removing them orphans no
// surviving character (see crdt.Text.Collectable). It returns nil when fewer than the
// configured threshold are present, so a sweep is only triggered once enough space is
// reclaimable to justify the broadcast.
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
// the edits that restart an Arc's quiescence window.
func containsTextEdit(actions []Action) bool {
	for _, a := range actions {
		if a.Type == ActionTypeInsertChar || a.Type == ActionTypeDeleteChar {
			return true
		}
	}
	return false
}

// Handle applies the set to the Arc's name or the graph's node inputs.
func (p SetPayload) Handle(state Arc) (Arc, error) {
	return edit(state, patch.Set(p))
}

// Handle applies the remove to the Arc's name or the graph's node inputs.
func (p RemovePayload) Handle(state Arc) (Arc, error) {
	return edit(state, patch.Remove(p))
}

// Handle applies the insert to the Arc's name or the graph's node inputs.
func (p InsertPayload) Handle(state Arc) (Arc, error) {
	return edit(state, patch.Insert(p))
}

// Handle applies the move to the Arc's name or the graph's node inputs.
func (p MovePayload) Handle(state Arc) (Arc, error) {
	return edit(state, patch.Move(p))
}

// document is the part of an Arc, outside its graph, that path actions edit.
type document struct {
	Name string `json:"name"`
}

// edit applies a path action to the Arc's name or to the inputs of the node that its
// path names. A path to another field is rejected.
func edit(state Arc, a patch.Action) (Arc, error) {
	if patch.Field(a) != "graph" {
		b := state.Base()
		d, err := patch.Edit(document{Name: b.Name}, a.Apply)
		if err != nil {
			return Arc{}, err
		}
		b.Name = d.Name
		state.SetBase(b)
		return state, nil
	}
	var err error
	state, editErr := onGraph(state, func(g *graph.Graph) {
		var inputs map[string]msgpack.EncodedJSON
		if inputs, err = patch.EditMap(
			g.Inputs,
			[]string{"graph", "inputs"},
			a,
			nil,
		); err == nil {
			g.Inputs = inputs
		}
	})
	if err != nil {
		return Arc{}, err
	}
	return state, editErr
}
