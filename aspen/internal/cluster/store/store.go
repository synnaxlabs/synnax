// Package store exposes a simple copy-on-read store for managing cluster state.
// SinkTarget create a new store, call store.New().
package store

import (
	"github.com/arya-analytics/aspen/internal/node"
	"github.com/arya-analytics/x/binary"
	"github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/store"
	"io"
)

// Store is an interface representing a copy-on-read store for managing cluster state.
type Store interface {
	// Observable allows the caller to react to state changes. This state is not diffed i.e.
	// any call that modifies the state, even if no actual change occurs, will get sent to the
	// Observable.
	store.Observable[State]
	// FlushLoader allows the caller to flush and load State from a persistent store.
	kv.FlushLoader
	// Set sets a node in state.
	Set(node.Node)
	// Get returns a node from state. Returns false if the node is not found.
	Get(node.ID) (node.Node, bool)
	// Merge merges a node.Group into State.Nodes by selecting nodes from group with heartbeats
	// that are either not in State or are older than in State.
	Merge(group node.Group)
	// Valid returns true if the store is valid i.e. State.HostID has been set.
	Valid() bool
	// GetHost returns the host node of the store.
	GetHost() node.Node
	// SetHost sets the host for the store.
	SetHost(node.Node)
}

func _copy(s State) State { return State{Nodes: s.Nodes.Copy(), HostID: s.HostID} }

// New opens a new empty, invalid Store.
func New() Store {
	c := &core{Observable: store.ObservableWrap[State](store.New(_copy))}
	c.Observable.SetState(State{Nodes: make(node.Group)})
	return c
}

// State is the current state of the cluster as viewed from the host.
type State struct {
	HostID node.ID
	Nodes  node.Group
}

func (s State) Flush(w io.Writer) error {
	if err := binary.Write(w, s.HostID); err != nil {
		return err
	}
	for _, n := range s.Nodes {
		if err := n.Flush(w); err != nil {
			return err
		}
	}
	return nil
}

func (s *State) Load(r io.Reader) error {
	s.Nodes = make(node.Group)
	if err := binary.Read(r, &s.HostID); err != nil {
		return err
	}
	for {
		n := &node.Node{}
		err := n.Load(r)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		s.Nodes[n.ID] = *n
	}
}

type core struct{ store.Observable[State] }

// Get implements Store.
func (c *core) Get(id node.ID) (node.Node, bool) {
	n, ok := c.Observable.ReadState().Nodes[id]
	return n, ok
}

// GetHost implements Store.
func (c *core) GetHost() node.Node {
	n, _ := c.Get(c.Observable.ReadState().HostID)
	return n
}

// SetHost implements Store.
func (c *core) SetHost(n node.Node) {
	snap := c.Observable.CopyState()
	snap.Nodes[n.ID] = n
	snap.HostID = n.ID
	c.Observable.SetState(snap)
}

// Set implements Store.
func (c *core) Set(n node.Node) {
	snap := c.Observable.CopyState()
	snap.Nodes[n.ID] = n
	c.Observable.SetState(snap)
}

// Merge implements Store.
func (c *core) Merge(other node.Group) {
	snap := c.Observable.CopyState()
	for _, n := range other {
		in, ok := snap.Nodes[n.ID]
		if !ok || n.Heartbeat.OlderThan(in.Heartbeat) {
			snap.Nodes[n.ID] = n
		}
	}
	c.Observable.SetState(snap)
}

// Valid implements Store.
func (c *core) Valid() bool { return c.GetHost().ID != 0 }

// Load implements kv.FlushLoader.
func (c *core) Load(r io.Reader) error {
	snap := &State{Nodes: make(node.Group)}
	if err := snap.Load(r); err != nil {
		return err
	}
	c.Observable.SetState(*snap)
	return nil
}

// Flush implements kv.FlushLoader.
func (c *core) Flush(w io.Writer) error { return c.Observable.CopyState().Flush(w) }
